export type ApiResponse<T> = {
  code: number;
  data: T;
  message: string;
};

export const AUTH_EXPIRED_EVENT = "traffic-monitor:auth-expired";

type QueryValue = string | number | boolean | null | undefined;

export function withQuery(path: string, params: Record<string, QueryValue>) {
  const search = new URLSearchParams();

  Object.entries(params).forEach(([key, value]) => {
    if (value === null || value === undefined || value === "") {
      return;
    }

    search.set(key, String(value));
  });

  const query = search.toString();
  return query ? `${path}?${query}` : path;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  let payload: ApiResponse<T> | null = null;
  try {
    payload = (await response.json()) as ApiResponse<T>;
  } catch {
    payload = null;
  }

  if (!response.ok) {
    if (response.status === 401 && payload?.message === "unauthorized") {
      if (typeof window !== "undefined") {
        window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT));
      }

      throw new Error("");
    }

    throw new Error(payload?.message || `request failed: ${response.status}`);
  }

  if (!payload) {
    throw new Error("empty response");
  }

  return payload.data;
}

export function get<T>(path: string) {
  return request<T>(path);
}

export function post<T, B = unknown>(path: string, body?: B) {
  return request<T>(path, {
    method: "POST",
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

export function put<T, B = unknown>(path: string, body: B) {
  return request<T>(path, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export function patch<T, B = unknown>(path: string, body: B) {
  return request<T>(path, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export function del<T>(path: string) {
  return request<T>(path, {
    method: "DELETE",
  });
}
