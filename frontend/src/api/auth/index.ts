import { AxiosPromise } from "axios";
import Endpoints from "../endpoints";
import { axiosInstance } from "../instance";
import { ILoginResponse, ILoginRequest } from "./types";

export const login = (params: ILoginRequest): AxiosPromise<ILoginResponse> =>
    axiosInstance.post(Endpoints.AUTH.LOGIN, params)

export const getServers = (): AxiosPromise<string> => axiosInstance.get(Endpoints.AUTH.GETSERVERS)

export const deleteServer = (serverUrl: string): AxiosPromise<void> => axiosInstance.delete(Endpoints.AUTH.DELETESERVERS,
    { data: { url: serverUrl } })
