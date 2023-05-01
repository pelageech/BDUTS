package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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

	subjectNewUser = "Your credentials for BDUTS load balancer"
	msgNewUser     = "Your username: %s\n" +
		"Your password: %s\n\n" +
		"Please log in and change your password.\n" +
		"By changing your temporary password, you're helping to ensure that your account is secure " +
		"and that only you have access to it. It's also an opportunity to choose a password " +
		"that's easy for you to remember, but difficult for others to guess."

	subjectPasswordChanged = "Your password has been changed"
	msgPasswordChanged     = "Your password has been changed.\nIf" +
		" you did not change your password, please contact the administrator."

	changePasswordError = "Password length must be between 10 and 25 characters. New password and new password confirmation must match."
)

type Service struct {
	db        db.Service
	sender    *email.Sender
	validator *validator.Validate
	signKey   []byte
	logger    *log.Logger
}

type userKey struct{}

func New(db db.Service, sender *email.Sender, validator *validator.Validate, signKey []byte, logger *log.Logger) *Service {
	return &Service{
		db:        db,
		sender:    sender,
		validator: validator,
		signKey:   signKey,
		logger:    logger,
	}
}

type SignUpUser struct {
	Username string `json:"username" validate:"required,min=4,max=20,alphanum"`
	Email    string `json:"email" validate:"required,email"`
}

type LogInUser struct {
	Username string `json:"username" validate:"required,min=4,max=20,alphanum"`
	Password string `json:"password" validate:"required"`
}

type ChangePasswordUser struct {
	OldPassword        string `json:"oldPassword" validate:"required"`
	NewPassword        string `json:"newPassword" validate:"required,min=10,max=25"`
	NewPasswordConfirm string `json:"newPasswordConfirm" validate:"required,eqfield=NewPassword"`
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
		s.logger.Error("Failed to generate salt", "err", err)
		return
	}
	salt = base64.URLEncoding.EncodeToString(saltBytes)
	return
}

func (s *Service) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var user SignUpUser
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		s.logger.Warn("Failed to decode SignUpUser", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.validator.Struct(user)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			s.logger.Warn("Failed validation of SignUpUser", "err", e)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
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

	errs := s.db.InsertUser(user.Username, salt, hashString, user.Email)
	if len(errs) != 0 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	msg := fmt.Sprintf(msgNewUser, user.Username, password)
	if err := s.sender.Send(
		user.Email,
		subjectNewUser,
		msg,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Service) generateToken(username string) (signedToken string, err error) {
	expirationTime := time.Now().Add(20 * time.Minute)

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
		return false
	}

	// Compare the password with the hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+salt))
	return err == nil
}

func (s *Service) SignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var user LogInUser
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.validator.Struct(user)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			s.logger.Warn("Error validating user", "err", e)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	if !s.isAuthorized(user.Username, user.Password) {
		w.WriteHeader(http.StatusUnauthorized)
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
		for _, e := range err.(validator.ValidationErrors) {
			s.logger.Warn("Error validating change password request", "err", e)
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(changePasswordError))
			return
		}
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

	err = s.sender.Send(emailAddress, subjectPasswordChanged, msgPasswordChanged)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authorizationHeader, " ")
		if len(bearerToken) != 2 {
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
