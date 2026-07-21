const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

function getToken() {
  return localStorage.getItem("token");
}

async function request(path, options = {}) {
  const headers = { "Content-Type": "application/json", ...options.headers };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${API_URL}${path}`, { ...options, headers });
  const isJson = res.headers.get("content-type")?.includes("application/json");
  const data = isJson ? await res.json() : null;

  if (!res.ok) {
    throw new Error(data?.error || `Request failed (${res.status})`);
  }
  return data;
}

export const api = {
  register: (email, password, fullName) =>
    request("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, full_name: fullName }),
    }),

  login: (email, password) =>
    request("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  submitVideo: (source, language) =>
    request("/api/v1/videos", {
      method: "POST",
      body: JSON.stringify({ source, language }),
    }),

  listVideos: () => request("/api/v1/videos"),

  getVideo: (id) => request(`/api/v1/videos/${id}`),

  cancelVideo: (id) => request(`/api/v1/videos/${id}/cancel`, { method: "POST" }),

  dashboardStats: () => request("/api/v1/dashboard/stats"),

  chatHistory: (videoId) => request(`/api/v1/videos/${videoId}/chat`),

  sendChat: (videoId, question) =>
    request(`/api/v1/videos/${videoId}/chat`, {
      method: "POST",
      body: JSON.stringify({ question }),
    }),
};

export { getToken };
