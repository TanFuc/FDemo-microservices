import { Module } from '@nestjs/common';
import { ClientsModule, Transport } from '@nestjs/microservices';
import { join } from 'path';
import { ProfileController } from './controllers/profile.controller';
import { InternalController } from './controllers/internal.controller';
import { ProfileService } from './services/profile.service';

@Module({
  imports: [
    ClientsModule.register([
      {
        name: 'AUTH_PACKAGE',
        transport: Transport.GRPC,
        options: {
          package: 'auth',
          protoPath: join(__dirname, '../../protos/auth.proto'),
          url: process.env.AUTH_GRPC_URL || '0.0.0.0:50051',
        },
      },
    ]),
  ],
  controllers: [ProfileController, InternalController],
  providers: [ProfileService],
  exports: [ProfileService],
})
export class ProfileModule {}
