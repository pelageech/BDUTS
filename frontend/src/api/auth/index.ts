import { AxiosPromise, AxiosResponse } from "axios";
import Endpoints from "../endpoints";
import { axiosInstance } from "../instance";
import { ILoginResponse, ILoginRequest } from "./types";

export const login = (params: ILoginRequest): AxiosPromise<ILoginResponse> =>
    axiosInstance.post(Endpoints.AUTH.LOGIN, params)

export const getServers = (): Promise<AxiosResponse<{ URL: string, HealthCheckTcpTimeout: number, MaximalRequests: number, Alive: boolean }[], any>> =>
    axiosInstance.get(Endpoints.AUTH.GET);

export const deleteServer = (serverUrl: string): AxiosPromise<void> => axiosInstance.delete(Endpoints.AUTH.DELETE,
    { data: { url: serverUrl } })

export const addServer = (url: string, healthCheckTcpTimeout: number, maximalRequests: number): AxiosPromise<void> => axiosInstance.post(Endpoints.AUTH.ADD,
    { url: url, healthCheckTcpTimeout: healthCheckTcpTimeout, maximalRequests: maximalRequests })

export const addUser = (username: string, email: string): AxiosPromise<void> => axiosInstance.post(Endpoints.AUTH.SIGNUP,
    { username: username, email: email })

export const clearCache = (): AxiosPromise<void> => axiosInstance.delete(Endpoints.AUTH.CLEAR)