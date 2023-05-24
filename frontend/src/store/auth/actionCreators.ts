import { Dispatch } from "@reduxjs/toolkit"
import api from "../../api"
import { ILoginRequest } from "../../api/auth/types"
import {
    loginStart,
    loginSucess,
    loginFailure,
    getServersStart,
    getServersSuccess,
    getServersFailure,
    logoutSuccess,
} from "./authReducer"
import Cookies from "js-cookie";
import { history } from '../../utils/history'


export const loginUser =
    (data: ILoginRequest) =>
        async (dispatch: Dispatch<any>): Promise<void> => {
            try {
                dispatch(loginStart());
                const res = await api.auth.login(data);
                const token = res.headers['authorization'];
                dispatch(loginSucess(token));
                dispatch(getServers());
                Cookies.set('token', token, { expires: 7 });
            } catch (e: any) {
                console.error(e)
                dispatch(loginFailure(e.message))
            }
        }

export const logoutUser =
    () =>
        async (dispatch: Dispatch): Promise<void> => {
            try {

                dispatch(logoutSuccess())
                Cookies.remove("token");
                history.push('/')
            } catch (e) {
                console.error(e)
            }
        }

export const restoreUser =
    (token: string) =>
        async (dispatch: Dispatch<any>): Promise<void> => {
            try {
                dispatch(loginStart());
                dispatch(loginSucess(token));
                dispatch(getServers());
            } catch (e: any) {
                console.error(e)
                dispatch(loginFailure(e.message))
            }
        }

export const getServers = () =>
    async (dispatch: Dispatch<any>): Promise<void> => {
        try {
            dispatch(getServersStart())
            const res  = await api.auth.getServers()
            dispatch(getServersSuccess(res.data))
        } catch (e: any) {
            console.error(e)
            dispatch(getServersFailure(e.message))
        }
    }

export const deleteServer = (serverUrl: string) =>
    async (): Promise<void> => {
        try {
            await api.auth.deleteServer(serverUrl)
        } catch (e: any) {
            console.error(e)
        }
    }

export const addServer = (arg: { url: string, healthCheckTcpTimeout: number, maximalRequests: number }) =>
    async (): Promise<void> => {
        try {
            await api.auth.addServer(arg.url, arg.healthCheckTcpTimeout, arg.maximalRequests);
        } catch (e: any) {
            console.error(e)
        }
    }

export const addUser = (arg: { username: string, email: string }) =>
    async (): Promise<void> => {
        try {
            await api.auth.addUser(arg.username, arg.email);
        } catch (e: any) {
            console.error(e)
        }
    }

export const changePassword = (arg: { oldPass: string, newPass: string }) =>
    async (): Promise<void> => {
        try {
            await api.changePass.changePassword(arg.oldPass, arg.newPass);
        } catch (e: any) {
            console.error(e)
        }
    }