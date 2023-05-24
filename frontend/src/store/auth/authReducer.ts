import { createSlice, PayloadAction } from '@reduxjs/toolkit'

export interface AuthState {
    authData: {
        accessToken: string | null,
        isLoading: boolean
        error: string | null,
    },
    serverData: {
        servers: { URL: string, HealthCheckTcpTimeout: number, MaximalRequests: number, Alive: boolean }[],
        urls: string[],
        isLoading: boolean
        error: string | null,
    },
}

const initialState: AuthState = {
    authData: {
        accessToken: null,
        isLoading: false,
        error: null,
    },
    serverData: {
        servers: [],
        urls: [],
        isLoading: false,
        error: null,
    },
}

export const authReducer = createSlice({
    name: 'auth',
    initialState,
    reducers: {
        loginStart: (state: AuthState): AuthState => ({
            ...state,
            authData: {
                ...state.authData,
                isLoading: true,
            }
        }),
        loginSucess: (state: AuthState, action: PayloadAction<string>): AuthState => ({
            ...state,
            authData: {
                ...state.authData,
                accessToken: action.payload,
                isLoading: false,
                error: null,
            }
        }),
        loginFailure: (state: AuthState, action: PayloadAction<string>): AuthState => ({
            ...state,
            authData: {
                ...state.authData,
                isLoading: false,
                error: action.payload,
            }
        }),
        getServersStart: (state: AuthState): AuthState => ({
            ...state,
            serverData: {
                ...state.serverData,
                isLoading: true,
            }
        }),
        getServersSuccess: (state: AuthState, action: PayloadAction<{ URL: string, HealthCheckTcpTimeout: number, MaximalRequests: number, Alive: boolean }[]>): AuthState => ({
            ...state,
            serverData: {
                ...state.serverData,
                servers: action.payload.map((serverObj) => {
                    return {
                        URL: serverObj.URL,
                        HealthCheckTcpTimeout: serverObj.HealthCheckTcpTimeout,
                        MaximalRequests: serverObj.MaximalRequests,
                        Alive: serverObj.Alive,
                    };
                }),
                urls: action.payload.map((serverObj) => serverObj.URL),
                isLoading: false,
                error: null,
            }
        }),
        getServersFailure: (state: AuthState, action: PayloadAction<string>): AuthState => ({
            ...state,
            serverData: {
                ...state.serverData,
                isLoading: false,
                error: action.payload,
            }
        }),
        logoutSuccess: (): AuthState => initialState,
    },
})

export const {
    loginStart,
    loginSucess,
    loginFailure,
    logoutSuccess,
    getServersStart,
    getServersSuccess,
    getServersFailure,
} = authReducer.actions

export default authReducer.reducer