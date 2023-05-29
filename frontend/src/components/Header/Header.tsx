import React from "react";
import { useSelector } from "react-redux";
import { Link } from "react-router-dom";
import { IRootState, useAppDispatch } from "../../store";
import "./Header.css";
import { logoutUser } from "../../store/auth/actionCreators";

const Header = () => {
    const isLoggedIn = useSelector(
        (state: IRootState) => !!state.auth.authData.accessToken
    );

    const dispatch = useAppDispatch();

    const handleLogout = () => {
        dispatch(logoutUser());
    };

    return (
        <nav className="header">
            {isLoggedIn && (
                <div className="button-container">
                    <Link to="/" className="button">
                        Dashboard
                    </Link>
                    <Link to="/signup" className="button">
                        Add user
                    </Link>
                    <Link to="/changepassword" className="button">
                        Change password
                    </Link>
                    <a href="https://v1722521.hosted-by-vdsina.ru:3000/" target="_blank" className="button">
                        Grafana
                    </a>
                    <button onClick={handleLogout}>
                        Logout
                    </button>
                </div>
            )}
        </nav>
    );
};

export default Header;
