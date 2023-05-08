import { Dispatch } from "@reduxjs/toolkit"
import api from "../../api"
import { ILoginRequest } from "../../api/auth/types"
import { loginStart, loginSucess, loginFailure } from "./authReducer"

export const loginUser =
    (data: ILoginRequest) =>
        async (dispatch: Dispatch<any>): Promise<void> => {
            try {
                dispatch(loginStart())
                const res = await api.auth.login(data)
                dispatch(loginSucess(res.headers['authorization']))
            } catch (e: any) {
                console.error(e)
                dispatch(loginFailure(e.message))
            }
        }


// export const getServers = () => {
//     return async dispatch => {
//         dispatch({ type: FETCH_SERVERS_REQUEST });

//         try {
//             const response = await fetch('https://api.example.com/servers');
//             const data = await response.json();
//             dispatch({ type: FETCH_SERVERS_SUCCESS, payload: data.servers });
//         } catch (error) {
//             dispatch({ type: FETCH_SERVERS_FAILURE, payload: error.message });
//         }
//     };
// };