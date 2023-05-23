import React, { FormEvent, useState } from "react";
import { useAppDispatch } from "../../store";
import { addUser } from "../../store/auth/actionCreators";
import "./SignUp.css";

const SignUp = () => {

    const dispatch = useAppDispatch();

    const [username, setUsername] = useState('');
    const [email, setEmail] = useState('');

    const handleSignUp = (e: FormEvent) => {
        e.preventDefault();
        dispatch(addUser({ username, email }));
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