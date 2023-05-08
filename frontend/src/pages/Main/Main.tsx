import React from "react";
import Login from "./components/Login";
import { useSelector } from "react-redux";
import { IRootState, useAppDispatch } from "../../store";
// import { getServers } from "../../store/auth/actionCreators";

const Main = () => {

    const dispatch = useAppDispatch()

    const isLoggedIn = useSelector(
        (state: IRootState) => !!state.auth.authData.accessToken
    );

    const renderProfile = () => (
        <div>
            <div>Вы успушно авторизовались</div>
            <button onClick={() => { window.location.reload() }}>Logout</button>
            {/* <button onClick={() => { dispatch(getServers()) }}>Get servers</button> */}
        </div>
    );

    return (
        <div>
            Main
            {isLoggedIn ? renderProfile() : <Login />}
        </div>
    );
};

export default Main;


