/** Configuration options for the HTTP client. */
export interface HttpConfig {
  /** Base URL for all requests. */
  baseUrl: string;
  /** Timeout in milliseconds. */
  timeout?: number;
  /** Custom headers. */
  headers?: Record<string, string>;
}

/** HTTP response object. */
export interface HttpResponse<T = any> {
  status: number;
  statusText: string;
  headers: Record<string, string>;
  data: T;
}

/** Error thrown when an HTTP request fails. */
export declare class HttpError extends Error {
  readonly status: number;
  readonly response: HttpResponse;
  constructor(message: string, status: number, response: HttpResponse);
}

/** HTTP client for making web requests. */
export declare class HttpClient {
  private config;
  constructor(config: HttpConfig);
  /** Perform a GET request. */
  get<T>(url: string, params?: Record<string, string>): Promise<HttpResponse<T>>;
  /** Perform a POST request. */
  post<T>(url: string, body: any): Promise<HttpResponse<T>>;
  /** Perform a PUT request. */
  put<T>(url: string, body: any): Promise<HttpResponse<T>>;
  /** Perform a DELETE request. */
  delete(url: string): Promise<HttpResponse<void>>;
  /** Close the client and release resources. */
  close(): void;
}

/** Supported HTTP methods. */
export declare enum HttpMethod {
  GET = "GET",
  POST = "POST",
  PUT = "PUT",
  DELETE = "DELETE",
  PATCH = "PATCH"
}

/** Maximum number of retries. */
export declare const MAX_RETRIES: number;

/** Create a pre-configured client with defaults. */
export declare function createClient(baseUrl: string): HttpClient;

/** Type alias for request interceptor. */
export type RequestInterceptor = (config: HttpConfig) => HttpConfig | Promise<HttpConfig>;
