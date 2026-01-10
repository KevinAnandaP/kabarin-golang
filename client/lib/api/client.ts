// API Client untuk komunikasi dengan backend

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class ApiClient {
  private baseUrl: string;

  constructor() {
    this.baseUrl = API_URL;
  }

  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const config: RequestInit = {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      credentials: 'include', // Untuk HTTP-Only Cookie
    };

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Request failed');
      }

      return response.json();
    } catch (error) {
      console.error('API Error:', error);
      throw error;
    }
  }

  // Auth endpoints
  async login(email: string, password: string) {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  }

  async register(email: string, password: string, name: string) {
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name }),
    });
  }

  async logout() {
    return this.request('/auth/logout', {
      method: 'POST',
    });
  }

  async getMe() {
    return this.request('/auth/me');
  }

  // Contact endpoints
  async addContact(uniqueId: string) {
    return this.request('/contacts', {
      method: 'POST',
      body: JSON.stringify({ uniqueId }),
    });
  }

  async getContacts() {
    return this.request('/contacts');
  }

  // Messages endpoints
  async getMessages(chatId: string, page = 1, limit = 50) {
    return this.request(`/messages/${chatId}?page=${page}&limit=${limit}`);
  }

  async sendMessage(receiverId: string, content: string) {
    return this.request('/messages', {
      method: 'POST',
      body: JSON.stringify({ receiverId, content }),
    });
  }
}

export const apiClient = new ApiClient();
