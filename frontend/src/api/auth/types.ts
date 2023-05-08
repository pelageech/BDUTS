// login

export interface ILoginRequest {
    username: string
    password: string
  }
  
  export interface ILoginResponse {
    accessToken: string
  }