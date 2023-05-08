import axios from 'axios'
import { store } from '../store'

export const axiosInstance = axios.create({ baseURL: 'https://195.2.93.80:8080' })

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