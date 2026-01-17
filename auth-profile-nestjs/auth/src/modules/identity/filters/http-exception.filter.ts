import {
  ExceptionFilter,
  Catch,
  ArgumentsHost,
  HttpException,
  HttpStatus,
  Logger,
} from '@nestjs/common';
import { Request, Response } from 'express';
import { RESPONSE_CODES, ERROR_MESSAGES } from '../constants';

interface ExceptionResponse {
  message: string | string[];
  error?: string;
  statusCode?: number;
}

interface ErrorResponse {
  success: false;
  code: string;
  message: string;
  errors: string[];
  timestamp: string;
  path: string;
}

@Catch()
export class HttpExceptionFilter implements ExceptionFilter {
  private readonly logger = new Logger(HttpExceptionFilter.name);

  catch(exception: unknown, host: ArgumentsHost): void {
    const ctx = host.switchToHttp();
    const response = ctx.getResponse<Response>();
    const request = ctx.getRequest<Request>();

    let status = HttpStatus.INTERNAL_SERVER_ERROR;
    let message: string = ERROR_MESSAGES.INTERNAL_ERROR;
    let errors: string[] = [];
    let code: string = RESPONSE_CODES.INTERNAL_ERROR;

    if (exception instanceof HttpException) {
      status = exception.getStatus();
      const exceptionResponse = exception.getResponse() as string | ExceptionResponse;

      if (typeof exceptionResponse === 'string') {
        message = exceptionResponse;
      } else {
        message =
          typeof exceptionResponse.message === 'string'
            ? exceptionResponse.message
            : exceptionResponse.message?.[0] || exception.message;

        errors = Array.isArray(exceptionResponse.message) ? exceptionResponse.message : [];
      }

      code = this.getErrorCode(status);
    } else if (exception instanceof Error) {
      message = exception.message || ERROR_MESSAGES.INTERNAL_ERROR;
      this.logger.error(`Unhandled exception: ${exception.message}`, exception.stack);
    } else {
      this.logger.error('Unknown exception type', String(exception));
    }

    const errorResponse: ErrorResponse = {
      success: false,
      code,
      message,
      errors: errors.length > 0 ? errors : [message],
      timestamp: new Date().toISOString(),
      path: request.url,
    };

    // Log error details (exclude sensitive info in production)
    if (status >= 500) {
      this.logger.error(
        `[${request.method}] ${request.url} - ${status}: ${message}`,
        exception instanceof Error ? exception.stack : undefined,
      );
    } else {
      this.logger.warn(`[${request.method}] ${request.url} - ${status}: ${message}`);
    }

    response.status(status).json(errorResponse);
  }

  private getErrorCode(status: number): string {
    switch (status) {
      case HttpStatus.BAD_REQUEST:
        return RESPONSE_CODES.BAD_REQUEST;
      case HttpStatus.UNAUTHORIZED:
        return RESPONSE_CODES.UNAUTHORIZED;
      case HttpStatus.FORBIDDEN:
        return RESPONSE_CODES.FORBIDDEN;
      case HttpStatus.NOT_FOUND:
        return RESPONSE_CODES.NOT_FOUND;
      case HttpStatus.CONFLICT:
        return RESPONSE_CODES.CONFLICT;
      default:
        return RESPONSE_CODES.INTERNAL_ERROR;
    }
  }
}
