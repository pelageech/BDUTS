import React, { FormEvent, useState } from "react";
import { useAppDispatch } from "../../store";
import { addUser } from "../../store/auth/actionCreators";
import "./SignUp.css";

const SignUp = () => {

    const dispatch = useAppDispatch();

    const [username, setUsername] = useState("");
    const [email, setEmail] = useState("");
    const [error, setError] = useState("");
    const [msg, setMsg] = useState("");


    const handleSignUp = (e: FormEvent) => {
        e.preventDefault();
        if (username === "" || email === ""){
            setError("Please fill in all fields");
            setTimeout(() => {
                setError("");
            }, 5000);
            return;
        }
        dispatch(addUser({ username, email }));
        setMsg("We've send a username and password to your email. Use them to login.");
        
    };

    const renderProfile = () => (
        <div className="container">
            <div className="form-container">
                <form onSubmit={handleSignUp}>
                    <div>
                        <label htmlFor="username">Username:</label>
                        <input type="text" value={username} onChange={(e) => setUsername(e.target.value)} />
                    </div>
                    <div>
                        <label htmlFor="email">Email:</label>
                        <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
                    </div>
                    {error && <span className="error-message">{error}</span>}
                    {msg && <span>{msg}</span>}
                    <button type="submit">Add user</button>
                </form>
            </div>
        </div>

    );

    return (
        <div>
            {renderProfile()}
        </div>
    );
};

export default SignUp;