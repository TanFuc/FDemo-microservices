import { Injectable, Logger, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { connect, NatsConnection, JetStreamClient, JetStreamManager, ConsumerConfig, AckPolicy, DeliverPolicy } from 'nats';
import { ProfileService } from '../profile/services/profile.service';

interface UserRegisteredEvent {
  userId: string;
  email: string;
  fullName: string;
  registeredAt: string;
}

@Injectable()
export class UserRegisteredListener implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(UserRegisteredListener.name);
  private natsConnection: NatsConnection | null = null;
  private isRunning = false;

  constructor(private readonly profileService: ProfileService) {}

  async onModuleInit() {
    try {
      await this.start();
    } catch (error) {
      this.logger.warn('Failed to start user registered listener', error);
    }
  }

  async onModuleDestroy() {
    await this.stop();
  }

  private async start(): Promise<void> {
    const natsUrl = process.env.NATS_URL || 'nats://localhost:4222';

    try {
      this.natsConnection = await connect({ servers: natsUrl });
      const js: JetStreamClient = this.natsConnection.jetstream();
      const jsm: JetStreamManager = await this.natsConnection.jetstreamManager();

      // Ensure stream exists
      try {
        await jsm.streams.info('USERS');
      } catch {
        // Stream will be created by auth service, wait for it
        this.logger.warn('USERS stream not found, waiting for auth service to create it');
        return;
      }

      // Create durable consumer
      const consumerConfig: Partial<ConsumerConfig> = {
        durable_name: 'profile-service-user-consumer',
        ack_policy: AckPolicy.Explicit,
        deliver_policy: DeliverPolicy.New,
        filter_subject: 'user.registered',
      };

      try {
        await jsm.consumers.add('USERS', consumerConfig);
      } catch {
        // Consumer might already exist
      }

      // Start consuming
      this.isRunning = true;
      this.consume(js);

      this.logger.log('User registered listener started');
    } catch (error) {
      this.logger.error('Failed to start user registered listener', error);
      throw error;
    }
  }

  private async consume(js: JetStreamClient): Promise<void> {
    try {
      const consumer = await js.consumers.get('USERS', 'profile-service-user-consumer');

      while (this.isRunning) {
        try {
          const messages = await consumer.fetch({ max_messages: 10, expires: 5000 });

          for await (const msg of messages) {
            try {
              const event: UserRegisteredEvent = JSON.parse(new TextDecoder().decode(msg.data));
              await this.handleUserRegistered(event);
              msg.ack();
            } catch (error) {
              this.logger.error('Failed to process message', error);
              msg.nak();
            }
          }
        } catch (error) {
          if (this.isRunning) {
            this.logger.error('Error fetching messages', error);
            await this.sleep(1000);
          }
        }
      }
    } catch (error) {
      this.logger.error('Consumer error', error);
    }
  }

  private async handleUserRegistered(event: UserRegisteredEvent): Promise<void> {
    this.logger.log(`Creating profile for user: ${event.userId}`);

    try {
      // Create initial profile with user info from auth service
      const profile = await this.profileService.createInitialProfile(
        event.userId,
        event.fullName,
        event.email,
      );

      this.logger.log(`Profile created for user: ${event.userId}, display name: ${profile.displayName}`);
    } catch (error) {
      this.logger.error(`Failed to create profile for user: ${event.userId}`, error);
      throw error; // Will cause NAK and retry
    }
  }

  private async stop(): Promise<void> {
    this.isRunning = false;
    if (this.natsConnection) {
      await this.natsConnection.drain();
      this.natsConnection = null;
      this.logger.log('User registered listener stopped');
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
