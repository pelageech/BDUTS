import { Dispatch } from "@reduxjs/toolkit"
import api from "../../api"
import cheerio from 'cheerio';
import { ILoginRequest } from "../../api/auth/types"
import { loginStart, loginSucess, loginFailure, loadProfileStart, loadProfileSucess, loadProfileFailure, loadServerStart, loadServerSucess, loadServerFailure } from "./authReducer"

export const loginUser =
    (data: ILoginRequest) =>
        async (dispatch: Dispatch<any>): Promise<void> => {
            try {
                dispatch(loginStart())
                const res = await api.auth.login(data)
                dispatch(loginSucess(res.headers['authorization']))
                dispatch(getServers())
            } catch (e: any) {
                console.error(e)
                dispatch(loginFailure(e.message))
            }
        }

export const getServers = () =>
    async (dispatch: Dispatch<any>): Promise<void> => {
        try {
            dispatch(loadServerStart())

            const res = await api.auth.getServers()

            dispatch(loadServerSucess(res.data))
        } catch (e: any) {
            console.error(e)

            dispatch(loadServerFailure(e.message))
        }
    }


export const deleteServer = (serverUrl: string) =>
    async (dispatch: Dispatch<any>): Promise<void> => {
        try {
            dispatch(loadServerStart())
            
            await api.auth.deleteServer(serverUrl)
            const res = await api.auth.getServers()
            dispatch(loadServerSucess(res.data))
        } catch (e: any) {
            console.error(e)

            dispatch(loadServerFailure(e.message))
        }
    }


export const htmlToMap = (html: string): { [key: string]: string } => {
    const $ = cheerio.load(html);
    const table = $('table');
    const serverMap: { [key: string]: string } = {};
    table.find('tr:gt(0)').each((i, row) => {
        const [id, url] = $(row).find('td').map((i, cell) => $(cell).text().trim()).get();
        serverMap[id] = url;
    });
    return serverMap;
};