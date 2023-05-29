import React, { FormEvent, useEffect, useState } from "react";
import Login from "./components/Login";
import { IRootState, useAppDispatch, useAppSelector } from "../../store";
import { addServer, deleteServer, getServers, clearCache } from "../../store/auth/actionCreators";
import "./Main.css";
import { getServersStart } from "../../store/auth/authReducer";

const Main = () => {
    const dispatch = useAppDispatch();

    const isLoggedIn = useAppSelector(
        (state: IRootState) => !!state.auth.authData.accessToken
    );

    const servers = useAppSelector(
        (state: IRootState) => state.auth.serverData.servers
    );

    useEffect(() => {
        dispatch(getServersStart());
    }, [dispatch]);

    const serverUrls = useAppSelector(
        (state: IRootState) => state.auth.serverData.urls
    );

    const [deleteUrl, setDeleteUrl] = useState("");
    const [url, setUrl] = useState("");
    const [healthCheckTcpTimeout, setHealthCheckTcpTimeout] = useState<number>(0);
    const [maximalRequests, setMaximalRequests] = useState<number>(0);
    const [deleteError, setDeleteError] = useState("");
    const [addError, setAddError] = useState("");


    const handleDelete = (e: FormEvent) => {
        e.preventDefault();
        if (serverUrls.includes(deleteUrl) && deleteUrl !== "") {
            dispatch(deleteServer(deleteUrl));
            dispatch(getServers());
            setDeleteUrl("");
        } else {
            setDeleteUrl("");
            setDeleteError("No URL found");
            setTimeout(() => {
                setDeleteError("");
            }, 5000);
            return;
        }
    };

    const handleAdd = (e: FormEvent) => {
        e.preventDefault();
        if (url === "" || healthCheckTcpTimeout === 0 || maximalRequests === 0) {
            setAddError("Please fill in all fields");
            setTimeout(() => {
                setAddError("");
            }, 5000);
            return;
        } else if (serverUrls.includes(url)) {
            setAddError("Server already exists");
            setTimeout(() => {
                setAddError("");
            }, 5000);
            return;
        }
        dispatch(addServer({ url, healthCheckTcpTimeout, maximalRequests }));
        dispatch(getServers());
        setUrl("");
        setMaximalRequests(0);
        setHealthCheckTcpTimeout(0);
    };

    const handleClearCache = (e: FormEvent) => {
        e.preventDefault();
        dispatch(clearCache());
    };

    const renderProfile = () => (
        <div className="profile-container">
            <table className="server-table">
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>URL</th>
                        <th>TCP Timeout</th>
                        <th>Max Requests</th>
                        <th>Alive</th>
                    </tr>
                </thead>
                <tbody>
                    {servers.map((server, index) => (
                        <tr key={index}>
                            <td>{index + 1}</td>
                            <td>{server.URL}</td>
                            <td>{server.HealthCheckTcpTimeout}</td>
                            <td>{server.MaximalRequests}</td>
                            <td>{server.Alive === true ? "true" : "false"}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
            <div>
                <form onSubmit={handleDelete}>
                    <label htmlFor="deleteServer">Server URL:</label>
                    <input
                        name="deleteServer"
                        type="text"
                        value={deleteUrl}
                        onChange={(e) => setDeleteUrl(e.target.value)}
                    />
                    <button>Delete</button>
                    {deleteError && <span className="error-message">{deleteError}</span>}
                </form>
            </div>
            <div>
                <form onSubmit={handleAdd}>
                    <label htmlFor="serverUrl">Server URL:</label>
                    <input
                        name="serverUrl"
                        type="text"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                    />
                    <label htmlFor="serverTcp">Server TCP:</label>
                    <input
                        name="serverTcp"
                        type="number"
                        value={healthCheckTcpTimeout}
                        onChange={(e) => setHealthCheckTcpTimeout(parseInt(e.target.value))}
                    />
                    <label htmlFor="serverMaxReq">Server MaxReq:</label>
                    <input
                        name="serverMaxReq"
                        type="number"
                        value={maximalRequests}
                        onChange={(e) => setMaximalRequests(parseInt(e.target.value))}
                    />
                    <button>Add</button>
                    {addError && <span className="error-message">{addError}</span>}
                </form>
            </div>
            <div>
                <form onSubmit={handleClearCache}>
                    <button>Clear cache</button>
                </form>
            </div>
        </div>
    );
    return <div className="main">{isLoggedIn ? renderProfile() : <Login />}</div>;
};

export default Main;