import React, { FormEvent, useState } from "react";
import { useAppDispatch } from "../../store";
import { changePassword } from "../../store/auth/actionCreators";
import "./ChangePassword.css";

const ChangePassword = () => {

    const dispatch = useAppDispatch();

    const [currentPassword, setCurrentPassword] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [error, setError] = useState("");

    const handleChangePassword = (e: FormEvent) => {
        e.preventDefault();
        if (currentPassword === "" || newPassword === "" || confirmPassword === "") {
            setError("Please fill in all fields");
            setTimeout(() => {
                setError("");
            }, 5000);
            return;
        } else if (newPassword !== confirmPassword) {
            setError("New password and confirmation password must match");
            setTimeout(() => {
                setError("");
            }, 5000);
            return;
        } else if (newPassword.length < 10 || newPassword.length > 25) {
            setError("Password length must be between 10 and 25 characters");
            setTimeout(() => {
                setError("");
            }, 5000);
            return;
        }
        const oldPass = currentPassword;
        const newPass = newPassword;
        dispatch(changePassword({ oldPass, newPass }))
    };


    const renderProfile = () => (
        <div className="container">
            <div className="form-container">
                <form onSubmit={handleChangePassword}>
                    <div>
                        <label htmlFor="currentPassword">Current password:</label>
                        <input type="password" id="currentPassword" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
                    </div>
                    <div>
                        <label htmlFor="newPassword">New password:</label>
                        <input type="password" id="newPassword" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
                    </div>
                    <div>
                        <label htmlFor="confirmPassword">Confirm password:</label>
                        <input type="password" id="confirmPassword" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} />
                    </div>
                    {error && <span className="error-message">{error}</span>}
                    <button type="submit">Change password</button>
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

export default ChangePassword;