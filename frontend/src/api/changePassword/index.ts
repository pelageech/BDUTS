import { AxiosPromise } from "axios";
import Endpoints from "../endpoints";
import { axiosInstance } from "../instance";

export const changePassword = (oldPass: string, newPass: string): AxiosPromise<void> => axiosInstance.patch(Endpoints.AUTH.PASSWORD,
    {"oldPassword":oldPass, "newPassword":newPass, "newPasswordConfirm":newPass})