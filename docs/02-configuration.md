# Configuration

GoStart applications are configured through a combination of a configuration file and environment variables. This provides a flexible way to manage settings across different environments (development, staging, production).

## Configuration Loading

The configuration is loaded by the `app.New()` function. It looks for a `.env` file in the root of the project and loads it into the environment. Then, it reads configuration values into a struct, allowing for type-safe access to your settings.

## Common Configuration Variables

Here are some of the standard environment variables used by GoStart:

- `APP_ENV`: The application environment (e.g., `development`, `production`).
- `APP_PORT`: The port on which the HTTP server will listen (e.g., `8080`).
- `LOG_LEVEL`: The level for logging (e.g., `debug`, `info`, `warn`, `error`).

### Database

- `DB_HOST`: The database host.
- `DB_PORT`: The database port.
- `DB_USER`: The database user.
- `DB_PASSWORD`: The database password.
- `DB_NAME`: The database name.

### Redis

- `REDIS_ADDR`: The address of the Redis server (e.g., `localhost:6379`).
- `REDIS_PASSWORD`: The password for the Redis server.

### AWS Cognito

- `AWS_REGION`: The AWS region where your Cognito User Pool is located.
- `AWS_COGNITO_USER_POOL_ID`: The ID of your Cognito User Pool.
- `AWS_COGNITO_CLIENT_ID`: The ID of your Cognito App Client.

## Adding Custom Configuration

To add your own configuration variables, you can extend the main configuration struct in `app/config.go` and load the corresponding environment variables.

1.  **Update the struct:**

    ```go
    // app/config.go
    type Config struct {
        // ... existing fields
        MyCustomSetting string `mapstructure:"MY_CUSTOM_SETTING"`
    }
    ```

2.  **Add the variable to your `.env` file:**

    ```
    MY_CUSTOM_SETTING=my_value
    ```

Your setting will now be available via the application's configuration object.
