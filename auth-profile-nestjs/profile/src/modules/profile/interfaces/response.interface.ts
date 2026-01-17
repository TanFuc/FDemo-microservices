export interface StandardResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

export interface PaginatedResponse<T> extends StandardResponse<T[]> {
  total: number;
  page: number;
  limit: number;
}
