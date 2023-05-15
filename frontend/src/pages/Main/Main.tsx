import React, { MouseEventHandler, useState } from "react";
import Login from "./components/Login";
import { useSelector } from "react-redux";
import { IRootState, useAppDispatch } from "../../store";
import { deleteServer } from "../../store/auth/actionCreators";

const Main = () => {

    const dispatch = useAppDispatch()

    const servers = useSelector(
        (state: IRootState) => state.auth.serverData.server
    );

    const isLoggedIn = useSelector(
        (state: IRootState) => !!state.auth.authData.accessToken
    );

    const serverIdUrl = useSelector(
        (state: IRootState) => state.auth.serverData.serverIdUrl
    );

    const [serverId, setServerId] = useState<number>(0);
    
    const handleClick: MouseEventHandler<HTMLButtonElement> = (event) => {
        const serverId = parseInt(event.currentTarget.dataset.serverid || '');
        const url = serverIdUrl[serverId];
        if (url) {
            dispatch(deleteServer(url));
            setServerId(0);
        } else {
            alert(`No URL found for ID ${serverId}`);
        }
    };

    const renderProfile = () => (
        <div>
            <div>Вы успушно авторизовались</div>
            <button onClick={() => { window.location.reload() }}>Logout</button>
            <div>
                <label htmlFor="serverIdInput">Server ID:</label>
                <input
                    id="serverIdInput"
                    type="number"
                    value={serverId}
                    onChange={(e) => setServerId(parseInt(e.target.value))}
                />
                <button onClick={handleClick} data-serverid={serverId}>Delete</button>
            </div>
            <button onClick={() => { }}>Add</button>
            <div id="html-container" dangerouslySetInnerHTML={{ __html: servers }} />
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


