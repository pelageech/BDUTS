import React, { useEffect } from "react";
import { BrowserRouter as Router, Routes, Route, Navigate } from "react-router-dom";
import Main from "./pages/Main";
import Header from "./components/Header/Header";
import { useSelector } from "react-redux";
import { IRootState, useAppDispatch } from "./store";
import SignUp from "./pages/SignUp/SignUp";
import ChangePassword from "./pages/ChangePassword";
import Cookies from "js-cookie";
import { restoreUser } from "./store/auth/actionCreators";

function App() {
    const dispatch = useAppDispatch();
    const isLoggedIn = useSelector((state: IRootState) => !!state.auth.authData.accessToken);

    useEffect(() => {
        const token = Cookies.get("token");
        if (token) {
            dispatch(restoreUser(token));
        }
    }, [dispatch]);
    
    return (
        <Router>
            <Header />
            <Routes>
                <Route path="/" element={<Main />} />
                <Route
                    path="/signup"
                    element={isLoggedIn ? <SignUp /> : <Navigate to="/" />}
                />
                <Route
                    path="/changepassword"
                    element={isLoggedIn ? <ChangePassword /> : <Navigate to="/" />}
                />
            </Routes>
        </Router>
    );
}

export default App;
