// Package auth implements a service for user authentication.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/charmbracelet/log"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"github.com/pelageech/BDUTS/db"
	"github.com/pelageech/BDUTS/email"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordLength = 25
	saltLength     = 20

	tokenExpirationTime = 20
	tokenLen            = 2

	changePasswordError   = "Password length must be between 10 and 25 characters. New password and new password confirmation must match."
	signUpValidationError = "Username must be between 4 and 20 characters. Username must contain only letters and numbers. Email must be valid."
	signInValidationError = "Username must be between 4 and 20 characters. Username must contain only letters and numbers. Password required."

	defaultUser     = "admin"
	defaultPassword = "verySecureAdminPassword12345"

	SignInError = "Invalid username or password"

	usernameKey = "username"
)

// Service is a service for user authentication.
type Service struct {
	db        *db.Service
	sender    *email.Sender
	validator *validator.Validate
	signKey   []byte
	logger    *log.Logger
}

type userKey struct{}

// New creates a new Service.
func New(db *db.Service, sender *email.Sender, validator *validator.Validate, signKey []byte, logger *log.Logger) *Service {
	return &Service{
		db:        db,
		sender:    sender,
		validator: validator,
		signKey:   signKey,
		logger:    logger,
	}
}

// SignUpUser is a user that signs up.
type SignUpUser struct {
	Username string `json:"username" validate:"required,min=4,max=20,alphanum"`
	Email    string `json:"email" validate:"required,email"`
}

// SignInUser is a user that logs in.
type SignInUser struct {
	Username string `json:"username" validate:"required,min=4,max=20,alphanum"`
	Password string `json:"password" validate:"required"`
}

// ChangePasswordUser is a user that changes password.
type ChangePasswordUser struct {
	OldPassword        string `json:"oldPassword" validate:"required"`
	NewPassword        string `json:"newPassword" validate:"required,min=10,max=25"`
	NewPasswordConfirm string `json:"newPasswordConfirm" validate:"required,eqfield=NewPassword"`
}

// AuthenticationMiddleware is a middleware that checks if the user is authenticated.
func (s *Service) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authorizationHeader, " ")
		if len(bearerToken) != tokenLen {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return s.signKey, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Name}))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !token.Valid || !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Store the authenticated user's username in the request context
		ctx := context.WithValue(r.Context(), userKey{}, claims["username"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) SignUpDefaultUser() error {
	salt, err := s.generateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword+salt), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Convert the hashed password to a string
	hashString := string(hash)

	err = s.db.InsertUser(defaultUser, salt, hashString, "admin@localhost")
	if err != nil {
		return fmt.Errorf("failed to insert default user: %w", err)
	}

	s.logger.Info("Default user created", "username", defaultUser, "password", defaultPassword)
	return nil
}

func (s *Service) generateRandomPassword() (password string, err error) {
	passwordBytes := make([]byte, passwordLength)
	_, err = rand.Read(passwordBytes)
	if err != nil {
		s.logger.Error("Failed to generate random password for new user", "err", err)
		return
	}
	password = base64.URLEncoding.EncodeToString(passwordBytes)
	return
}

func (s *Service) generateSalt() (salt string, err error) {
	saltBytes := make([]byte, saltLength)
	_, err = rand.Read(saltBytes)
	if err != nil {
		return
	}
	salt = base64.URLEncoding.EncodeToString(saltBytes)
	return
}

// SignUp signs up a user.
func (s *Service) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var user SignUpUser
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.validator.Struct(user)
	if err != nil {
		s.logger.Warn("Failed validation of SignUpUser", "err", err)
		http.Error(w, signUpValidationError, http.StatusBadRequest)
		return
	}

	password, err := s.generateRandomPassword()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	salt, err := s.generateSalt()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Error hashing password with salt", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Convert the hashed password to a string
	hashString := string(hash)

	err = s.db.InsertUser(user.Username, salt, hashString, user.Email)
	if err != nil {
		if errors.Is(err, bolt.ErrBucketExists) {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := s.sender.SendSignUpEmail(user.Email, user.Username, password); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Service) generateToken(username string) (signedToken string, err error) {
	expirationTime := time.Now().Add(tokenExpirationTime * time.Minute)

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"username": username,
		"exp":      expirationTime.Unix(),
	})

	signedToken, err = token.SignedString(s.signKey)
	return
}

func (s *Service) isAuthorized(username, password string) bool {
	salt, hash, err := s.db.GetSaltAndHash(username)
	if err != nil {
		s.logger.Warn("Error getting salt and hash", "username", username, "err", err)
		return false
	}

	// Compare the password with the hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+salt))
	if err != nil {
		s.logger.Warn("Error comparing password and hash", "username", username, "err", err)
		return false
	}
	return true
}

// SignIn signs in a user.
func (s *Service) SignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var user SignInUser
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.validator.Struct(user)
	if err != nil {
		s.logger.Warn("Error validating user", "err", err)
		http.Error(w, signInValidationError, http.StatusBadRequest)
		return
	}

	if !s.isAuthorized(user.Username, user.Password) {
		http.Error(w, SignInError, http.StatusUnauthorized)
		return
	}

	token, err := s.generateToken(user.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

// ChangePassword changes a user's password.
func (s *Service) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var user ChangePasswordUser
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		s.logger.Warn("Failed to decode change password request", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.validator.Struct(user)
	if err != nil {
		s.logger.Warn("Error validating change password request", "err", err)
		http.Error(w, changePasswordError, http.StatusUnprocessableEntity)
		return
	}

	username, ok := r.Context().Value(userKey{}).(string)
	if !ok {
		s.logger.Warn("Failed to get username from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !s.isAuthorized(username, user.OldPassword) {
		s.logger.Warn("User is not authorized to change password", "user", username)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	salt, err := s.generateSalt()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.NewPassword+salt), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Error hashing password", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hashString := string(hash)

	err = s.db.ChangePassword(username, salt, hashString)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	emailAddress, err := s.db.GetEmail(username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.sender.SendChangedPasswordEmail(emailAddress)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user := r.URL.Query().Get(usernameKey)

	err := s.db.DeleteUser(user)
	if err != nil {
		if errors.Is(err, bolt.ErrBucketNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Error("Error deleting user", "err", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
