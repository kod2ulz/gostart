# Authentication

Authentication in GoStart is designed to be robust and secure, while also being flexible enough to adapt to different requirements. The default implementation uses **AWS Cognito** with **JWTs (JSON Web Tokens)**.

## How it Works

1.  **Login:** A user authenticates with AWS Cognito (e.g., through a frontend application using the Amplify library). Cognito returns a set of JWTs (ID Token, Access Token, Refresh Token).

2.  **API Requests:** The client sends the **ID Token** in the `Authorization` header of every request to your GoStart API.

    ```
    Authorization: Bearer <your_id_token>
    ```

3.  **Middleware Verification:** The `api.AuthMiddleware()` intercepts the incoming request. It validates the JWT to ensure it is correctly signed by AWS Cognito and has not expired.

4.  **User Context:** If the token is valid, the middleware extracts the user's information (like user ID, email, etc.) from the token's claims and attaches it to the `api.Context`. 

5.  **Accessing the User:** In your API handlers, you can then access the authenticated user's data by calling `c.User()`.

## Protecting Routes

To protect a route or a group of routes, simply apply the `api.AuthMiddleware()`.

```go
// This group of routes requires a valid JWT
authRequired := app.Router.Group("/api/v1")
authRequired.Use(api.AuthMiddleware())

{
    authRequired.GET("/me", func(c *api.Context) {
        // The user is guaranteed to be available here
        user := c.User()
        c.Success(user)
    })
}
```

## The User Object

The `c.User()` method returns a `services.User` object (or a similar session object). This object contains the claims extracted from the JWT, such as:

- `ID`: The unique identifier for the user (the `sub` claim).
- `Email`: The user's email address.
- `Username`: The user's username.

## Configuration

For the Cognito integration to work, you must provide the following environment variables:

- `AWS_REGION`: The AWS region of your Cognito User Pool.
- `AWS_COGNITO_USER_POOL_ID`: The ID of the User Pool.
- `AWS_COGNITO_CLIENT_ID`: The App Client ID (optional in some verification flows but good practice to have).

## Extending Authentication

The authentication system is built around interfaces. You can create your own implementation of the `auth.Provider` interface to support other authentication methods, such as: 

- Other OAuth providers (Google, GitHub).
- Simple API keys.
- Session cookies.
