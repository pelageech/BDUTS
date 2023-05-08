import React from "react";
import { BrowserRouter as Router, Routes, Route, Navigate } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Main from "./pages/Main";
import Header from "./components/Header/Header";
import { useSelector } from "react-redux";
import { IRootState } from "./store";

function App() {
    const isLoggedIn = useSelector(
        (state: IRootState) => !!state.auth.authData.accessToken
    );

    return (
        <Router>
            <Header />
            <Routes>
                <Route path="/" element={<Main />} />
                <Route
                    path="/dashboard"
                    element={isLoggedIn ? <Dashboard /> : <Navigate to="/" />}
                />
            </Routes>
        </Router>

    );
}

export default App;
