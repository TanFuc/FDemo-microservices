import { Injectable, Logger, OnModuleDestroy, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { connect, NatsConnection, JetStreamClient, JetStreamManager } from 'nats';

export interface UserRegisteredEvent {
  userId: string;
  email: string;
  fullName: string;
  registeredAt: string;
}

@Injectable()
export class UserEventsService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(UserEventsService.name);
  private natsConnection: NatsConnection | null = null;
  private jetstream: JetStreamClient | null = null;
  private isConnected = false;

  constructor(private readonly configService: ConfigService) {}

  async onModuleInit() {
    try {
      await this.connect();
    } catch (error) {
      this.logger.warn('Failed to connect to NATS, events will not be published', error);
    }
  }

  async onModuleDestroy() {
    await this.disconnect();
  }

  private async connect(): Promise<void> {
    const natsUrl = this.configService.get<string>('NATS_URL', 'nats://localhost:4222');

    try {
      this.natsConnection = await connect({ servers: natsUrl });
      this.jetstream = this.natsConnection.jetstream();

      // Ensure stream exists
      const jsm: JetStreamManager = await this.natsConnection.jetstreamManager();

      try {
        await jsm.streams.info('USERS');
      } catch {
        // Create stream if it doesn't exist
        await jsm.streams.add({
          name: 'USERS',
          subjects: ['user.*'],
        });
        this.logger.log('Created USERS stream');
      }

      this.isConnected = true;
      this.logger.log(`Connected to NATS at ${natsUrl}`);
    } catch (error) {
      this.logger.error('Failed to connect to NATS', error);
      throw error;
    }
  }

  private async disconnect(): Promise<void> {
    if (this.natsConnection) {
      await this.natsConnection.drain();
      this.natsConnection = null;
      this.jetstream = null;
      this.isConnected = false;
      this.logger.log('Disconnected from NATS');
    }
  }

  async publishUserRegistered(event: UserRegisteredEvent): Promise<void> {
    if (!this.isConnected || !this.jetstream) {
      this.logger.warn('NATS not connected, skipping event publish');
      return;
    }

    try {
      const data = JSON.stringify(event);
      await this.jetstream.publish('user.registered', new TextEncoder().encode(data));
      this.logger.log(`Published user.registered event for user: ${event.userId}`);
    } catch (error) {
      this.logger.error('Failed to publish user.registered event', error);
      // Don't throw - registration should succeed even if event fails
    }
  }
}
