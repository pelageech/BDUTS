import axios from 'axios'
import { store } from '../store'

export const axiosInstance = axios.create({ baseURL: 'https://v1722521.hosted-by-vdsina.ru:8080' })

axiosInstance.interceptors.request.use(
    (config) => {
        const token = store.getState().auth.authData.accessToken;
        if (token) {
            config.headers.Authorization = `${token}`;
        }
        return config;
    },
    (error) => Promise.reject(error)
);
