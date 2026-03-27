namespace Example.Services;

/// <summary>
/// Configuration for the HTTP service.
/// </summary>
public class HttpConfig
{
    /// <summary>Base URL for all requests.</summary>
    public string BaseUrl { get; set; } = "";

    /// <summary>Timeout in milliseconds.</summary>
    public int Timeout { get; set; } = 30000;

    /// <summary>Whether to verify SSL certificates.</summary>
    public bool VerifySsl { get; set; } = true;
}

/// <summary>
/// Represents an HTTP response.
/// </summary>
public class HttpResponse<T>
{
    public int StatusCode { get; init; }
    public string StatusText { get; init; } = "";
    public Dictionary<string, string> Headers { get; init; } = new();
    public T? Data { get; init; }
}

/// <summary>
/// Thrown when an HTTP request fails.
/// </summary>
public class HttpException : Exception
{
    public int StatusCode { get; }
    public HttpResponse<string>? Response { get; }

    public HttpException(string message, int statusCode, HttpResponse<string>? response = null)
        : base(message)
    {
        StatusCode = statusCode;
        Response = response;
    }
}

/// <summary>
/// Interface for HTTP clients.
/// </summary>
public interface IHttpClient : IDisposable
{
    /// <summary>Perform a GET request.</summary>
    Task<HttpResponse<T>> GetAsync<T>(string url, CancellationToken ct = default);

    /// <summary>Perform a POST request.</summary>
    Task<HttpResponse<T>> PostAsync<T>(string url, object body, CancellationToken ct = default);

    /// <summary>Set a default header.</summary>
    void SetHeader(string name, string value);
}

/// <summary>
/// HTTP client implementation with retry support.
/// </summary>
public class HttpClient : IHttpClient
{
    private readonly HttpConfig _config;
    private readonly Dictionary<string, string> _headers = new();
    private bool _disposed;

    /// <summary>Maximum number of retries.</summary>
    public const int MaxRetries = 3;

    /// <summary>Current number of active requests.</summary>
    public int ActiveRequests { get; private set; }

    public HttpClient(HttpConfig config)
    {
        _config = config;
    }

    /// <summary>Perform a GET request.</summary>
    public async Task<HttpResponse<T>> GetAsync<T>(string url, CancellationToken ct = default)
    {
        throw new NotImplementedException();
    }

    /// <summary>Perform a POST request.</summary>
    public async Task<HttpResponse<T>> PostAsync<T>(string url, object body, CancellationToken ct = default)
    {
        throw new NotImplementedException();
    }

    /// <summary>Set a default header for all subsequent requests.</summary>
    public void SetHeader(string name, string value)
    {
        _headers[name] = value;
    }

    public void Dispose()
    {
        _disposed = true;
    }
}

/// <summary>
/// Supported HTTP methods.
/// </summary>
public enum HttpMethod
{
    Get,
    Post,
    Put,
    Delete,
    Patch
}

/// <summary>
/// Callback for request interception.
/// </summary>
public delegate Task<HttpConfig> RequestInterceptor(HttpConfig config);

/// <summary>
/// Result of a health check.
/// </summary>
public readonly struct HealthStatus
{
    public bool IsHealthy { get; init; }
    public string Message { get; init; }
    public TimeSpan Latency { get; init; }
}
