import {
  Injectable,
  NestInterceptor,
  ExecutionContext,
  CallHandler,
} from '@nestjs/common';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { Request } from 'express';
import { ApiResponse } from '../interfaces';
import { RESPONSE_CODES } from '../constants';

interface TransformableResponse<T> {
  data: T;
  message?: string;
  code?: string;
}

@Injectable()
export class ResponseInterceptor<T>
  implements NestInterceptor<T, ApiResponse<T>>
{
  intercept(
    context: ExecutionContext,
    next: CallHandler,
  ): Observable<ApiResponse<T>> {
    const request = context.switchToHttp().getRequest<Request>();
    const response = context.switchToHttp().getResponse();
    const statusCode = response.statusCode;

    return next.handle().pipe(
      map((responseData: T | TransformableResponse<T>) => {
        // If already formatted, pass through
        if (this.isApiResponse(responseData)) {
          return responseData as ApiResponse<T>;
        }

        // Handle response with nested data/message structure
        if (this.isTransformableResponse(responseData)) {
          return {
            success: true,
            code: responseData.code || this.getResponseCode(statusCode),
            message: responseData.message || 'Success',
            data: responseData.data,
            timestamp: new Date().toISOString(),
            path: request.url,
          };
        }

        // Default transformation
        return {
          success: true,
          code: this.getResponseCode(statusCode),
          message: 'Success',
          data: responseData,
          timestamp: new Date().toISOString(),
          path: request.url,
        };
      }),
    );
  }

  private isApiResponse(response: unknown): response is ApiResponse<T> {
    return (
      typeof response === 'object' &&
      response !== null &&
      'success' in response &&
      'code' in response &&
      'data' in response &&
      'timestamp' in response
    );
  }

  private isTransformableResponse(
    response: unknown,
  ): response is TransformableResponse<T> {
    return (
      typeof response === 'object' &&
      response !== null &&
      'data' in response &&
      !('success' in response)
    );
  }

  private getResponseCode(statusCode: number): string {
    if (statusCode === 201) return RESPONSE_CODES.CREATED;
    if (statusCode >= 200 && statusCode < 300) return RESPONSE_CODES.SUCCESS;
    return RESPONSE_CODES.SUCCESS;
  }
}
