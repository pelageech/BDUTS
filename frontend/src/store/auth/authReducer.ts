import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import cheerio from 'cheerio';

export interface AuthState {
    authData: {
        accessToken: string | null,
        isLoading: boolean
        error: string | null,
    },
    serverData: {
        serverHtml: string,
        serverUrl: string[],
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
        serverHtml: '<h1>There are no backends</h1>',
        serverUrl: [],
        isLoading: false,
        error: null,
    },
}

export const authReducer = createSlice({
    name: 'auth',
    initialState,
    reducers: {
        loginStart: (state): AuthState => ({
            ...state,
            authData: {
                ...state.authData,
                isLoading: true,
            }
        }),
        loginSucess: (state, action: PayloadAction<string>): AuthState => ({
            ...state,
            authData: {
                ...state.authData,
                accessToken: action.payload,
                isLoading: false,
                error: null,
            }
        }),
        loginFailure: (state, action: PayloadAction<string>): AuthState => ({
            ...state,
            authData: {
                ...state.authData,
                isLoading: false,
                error: action.payload,
            }
        }),
        getServersStart: (state): AuthState => ({
            ...state,
            serverData: {
                ...state.serverData,
                isLoading: true,
            }
        }),
        getServersSuccess: (state, action: PayloadAction<string>): AuthState => ({
            ...state,
            serverData: {
                ...state.serverData,
                serverHtml: action.payload,
                serverUrl: parseUrlsFromHtml(action.payload),
                isLoading: false,
                error: null,
            }
        }),
        getServersFailure: (state, action: PayloadAction<string>): AuthState => ({
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

function parseUrlsFromHtml(html: string): string[] {
    const $ = cheerio.load(html);
    const urls: string[] = [];
    $('table tr').each((_index: number, element: cheerio.Element) => {
      const url = $(element).find('td:nth-child(2)').text();
      urls.push(url);
    });
    return urls;
  }

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