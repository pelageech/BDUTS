import React, { FormEvent, useState } from "react";
import { IRootState, useAppDispatch, useAppSelector } from "../../../store";
import { loginUser } from "../../../store/auth/actionCreators";
import "./Login.css";
import Cookies from "js-cookie";
import { wait } from "@testing-library/user-event/dist/utils";

const Login = () => {
    const dispatch = useAppDispatch();

    const token = useAppSelector(
        (state: IRootState) => state.auth.authData.accessToken
    );

    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState("");

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();
        if (!username || !password) {
            setError("Input fields cannot be empty");
            return;
        }
        dispatch(loginUser({ username, password }));
        setTimeout(() => {
            if (token !== "") {
              setError("No such user");
            }
          }, 1000);
    };

    return (
        <div className="login-container">
            <form className="login-form" onSubmit={handleSubmit}>
                <div className="form-group">
                    <label htmlFor="login">Login:</label>
                    <input
                        className="input-field"
                        name="login"
                        type="text"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                    />
                </div>
                <div className="form-group">
                    <label htmlFor="password">Password:</label>
                    <input
                        className="input-field"
                        name="password"
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                    />
                </div>
                {error && <p className="error-message">{error}</p>}
                <button className="submit-button">Submit</button>
            </form>
        </div>
    );
};

export default Login;
