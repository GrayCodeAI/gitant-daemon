export interface Status {
  version: string;
  peers: number;
  repos: number;
  agents: number;
  uptime: string;
  identity: string;
}

export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  avatar_url: string;
  role: string;
  created_at: string;
}

export interface Repo {
  id: string;
  name: string;
  description: string;
  private: boolean;
  created_at: string;
}

export interface Issue {
  id: string;
  title: string;
  body: string;
  status: string;
  author: string;
  labels: string[];
  assignee: string;
  created_at: string;
  updated_at: string;
}

export interface PullRequest {
  id: string;
  title: string;
  body: string;
  status: string;
  author: string;
  source_branch: string;
  target_branch: string;
  labels: string[];
  assignee: string;
  reviewers: string[];
  created_at: string;
  updated_at: string;
}

export interface Discussion {
  id: string;
  title: string;
  body: string;
  author: string;
  category: string;
  status: string;
  tags: string[];
  answers: number;
  upvotes: number;
  views: number;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  status: string;
  columns: number;
  created_at: string;
  updated_at: string;
}

export interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  read: boolean;
  metadata: Record<string, any>;
  created_at: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  offset: number;
  limit: number;
}

export class GitantClient {
  private baseUrl: string;
  private token: string;

  constructor(baseUrl: string, token: string = '') {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.token = token;
  }

  setToken(token: string): void {
    this.token = token;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: any
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`API error (${response.status}): ${error}`);
    }

    return response.json();
  }

  // Status
  async getStatus(): Promise<Status> {
    return this.request('GET', '/api/v1/status');
  }

  async getHealth(): Promise<Record<string, any>> {
    return this.request('GET', '/health');
  }

  // Auth
  async register(
    username: string,
    email: string,
    password: string
  ): Promise<{ user: User; token: string }> {
    return this.request('POST', '/api/v1/auth/register', {
      username,
      email,
      password,
    });
  }

  async login(
    username: string,
    password: string
  ): Promise<{ user: User; token: string }> {
    return this.request('POST', '/api/v1/auth/login', {
      username,
      password,
    });
  }

  async getProfile(): Promise<User> {
    return this.request('GET', '/api/v1/auth/profile');
  }

  // Repos
  async listRepos(): Promise<Repo[]> {
    const result = await this.request<{ repos: Repo[] }>(
      'GET',
      '/api/v1/repos'
    );
    return result.repos;
  }

  async createRepo(
    name: string,
    description: string,
    private_: boolean = false
  ): Promise<Repo> {
    return this.request('POST', '/api/v1/repos', {
      name,
      description,
      private: private_,
    });
  }

  async getRepo(id: string): Promise<Repo> {
    return this.request('GET', `/api/v1/repos/${id}`);
  }

  async deleteRepo(id: string): Promise<void> {
    await this.request('DELETE', `/api/v1/repos/${id}`);
  }

  // Issues
  async listIssues(
    repoId: string,
    status?: string
  ): Promise<Issue[]> {
    const params = status ? `?status=${status}` : '';
    const result = await this.request<{ issues: Issue[] }>(
      'GET',
      `/api/v1/repos/${repoId}/issues${params}`
    );
    return result.issues;
  }

  async createIssue(
    repoId: string,
    title: string,
    body: string,
    labels: string[] = []
  ): Promise<Issue> {
    return this.request('POST', `/api/v1/repos/${repoId}/issues`, {
      title,
      body,
      labels,
    });
  }

  async closeIssue(repoId: string, issueId: string): Promise<void> {
    await this.request(
      'POST',
      `/api/v1/repos/${repoId}/issues/${issueId}/close`
    );
  }

  // Pull Requests
  async listPRs(
    repoId: string,
    status?: string
  ): Promise<PullRequest[]> {
    const params = status ? `?status=${status}` : '';
    const result = await this.request<{ prs: PullRequest[] }>(
      'GET',
      `/api/v1/repos/${repoId}/prs${params}`
    );
    return result.prs;
  }

  async createPR(
    repoId: string,
    title: string,
    body: string,
    sourceBranch: string,
    targetBranch: string
  ): Promise<PullRequest> {
    return this.request('POST', `/api/v1/repos/${repoId}/prs`, {
      title,
      body,
      source_branch: sourceBranch,
      target_branch: targetBranch,
    });
  }

  async mergePR(repoId: string, prId: string): Promise<void> {
    await this.request(
      'POST',
      `/api/v1/repos/${repoId}/prs/${prId}/merge`
    );
  }

  // Discussions
  async listDiscussions(
    repoId: string,
    category?: string,
    status?: string
  ): Promise<Discussion[]> {
    const params = new URLSearchParams();
    if (category) params.set('category', category);
    if (status) params.set('status', status);
    const query = params.toString() ? `?${params.toString()}` : '';
    const result = await this.request<{ discussions: Discussion[] }>(
      'GET',
      `/api/v1/repos/${repoId}/discussions${query}`
    );
    return result.discussions;
  }

  async createDiscussion(
    repoId: string,
    title: string,
    body: string,
    category: string = 'general',
    tags: string[] = []
  ): Promise<Discussion> {
    return this.request('POST', `/api/v1/repos/${repoId}/discussions`, {
      title,
      body,
      category,
      tags,
    });
  }

  // Projects
  async listProjects(repoId: string): Promise<Project[]> {
    const result = await this.request<{ projects: Project[] }>(
      'GET',
      `/api/v1/repos/${repoId}/projects`
    );
    return result.projects;
  }

  async createProject(
    repoId: string,
    name: string,
    description: string
  ): Promise<Project> {
    return this.request('POST', `/api/v1/repos/${repoId}/projects`, {
      name,
      description,
    });
  }

  // Notifications
  async listNotifications(
    unreadOnly: boolean = false
  ): Promise<{ notifications: Notification[]; unread_count: number }> {
    const params = unreadOnly ? '?unread=true' : '';
    return this.request(
      'GET',
      `/api/v1/notifications${params}`
    );
  }

  async markNotificationRead(id: string): Promise<void> {
    await this.request(
      'PUT',
      `/api/v1/notifications/${id}/read`
    );
  }

  async markAllNotificationsRead(): Promise<void> {
    await this.request('PUT', '/api/v1/notifications/read-all');
  }

  // Users
  async listUsers(): Promise<User[]> {
    const result = await this.request<{ users: User[] }>(
      'GET',
      '/api/v1/users'
    );
    return result.users;
  }

  async getUser(id: string): Promise<User> {
    return this.request('GET', `/api/v1/users/${id}`);
  }
}
