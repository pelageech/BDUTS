import React, { FormEvent, useState } from "react";
import { useAppDispatch } from "../../../store";
import { loginUser } from "../../../store/auth/actionCreators";

const Login = () => {
    const dispatch = useAppDispatch();

    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();

        dispatch(loginUser({ username, password }));
    };

    return (
        <div>
            <form onSubmit={handleSubmit}>
                <div>
                    <label htmlFor="login">Login:</label>
                    <input name="login" type="text" value={username} onChange={(e) => setUsername(e.target.value)} />
                </div>
                <div>
                    <label htmlFor="password">Password:</label>
                    <input name="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
                </div>
                <button>Submit</button>
            </form>
        </div>
    );
};

export default Login;