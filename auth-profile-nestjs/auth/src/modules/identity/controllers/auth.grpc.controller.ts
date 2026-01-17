import { Controller } from '@nestjs/common';
import { GrpcMethod } from '@nestjs/microservices';
import { AuthService } from '../services/auth.service';

@Controller()
export class AuthGrpcController {
  constructor(private readonly authService: AuthService) {}

  @GrpcMethod('AuthService', 'Authorize')
  async authorize(data: { userId: string; resource: string; action: string }) {
    return this.authService.authorize(data.userId, data.resource, data.action);
  }
}
