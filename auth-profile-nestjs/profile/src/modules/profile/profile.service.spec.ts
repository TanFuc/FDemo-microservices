import { ConflictException, NotFoundException } from '@nestjs/common';
import { Test, TestingModule } from '@nestjs/testing';
import { PrismaService } from '../../database/prisma.service';
import { AddressType } from './schemas/address.schema';
import { ProfileService } from './services/profile.service';

describe('ProfileService', () => {
  let service: ProfileService;

  const prisma = {
    profile: {
      findUnique: jest.fn(),
      create: jest.fn(),
      update: jest.fn(),
    },
    address: {
      count: jest.fn(),
      create: jest.fn(),
      findMany: jest.fn(),
      findFirst: jest.fn(),
      updateMany: jest.fn(),
      update: jest.fn(),
      delete: jest.fn(),
    },
    shopConfig: {
      findFirst: jest.fn(),
      create: jest.fn(),
      update: jest.fn(),
    },
    $transaction: jest.fn(),
  };

  const profile = {
    id: 'profile-1',
    userId: 'test-user-id',
    displayName: 'Test User',
    shopConfig: null,
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        ProfileService,
        {
          provide: PrismaService,
          useValue: prisma,
        },
      ],
    }).compile();

    service = module.get(ProfileService);
    jest.clearAllMocks();
    prisma.profile.findUnique.mockResolvedValue(profile);
  });

  it('returns an existing profile', async () => {
    await expect(service.getOrCreateProfile(profile.userId)).resolves.toEqual(
      profile,
    );
    expect(prisma.profile.findUnique).toHaveBeenCalledWith({
      where: { userId: profile.userId },
      include: { shopConfig: true },
    });
  });

  it('creates a profile when one does not exist', async () => {
    prisma.profile.findUnique.mockResolvedValue(null);
    prisma.profile.create.mockResolvedValue(profile);

    await expect(service.getOrCreateProfile(profile.userId)).resolves.toEqual(
      profile,
    );
    expect(prisma.profile.create).toHaveBeenCalledWith({
      data: {
        userId: profile.userId,
        displayName: 'User-r-id',
      },
      include: { shopConfig: true },
    });
  });

  it('makes the first address the default address', async () => {
    const dto = {
      contactName: 'John Doe',
      phone: '0912345678',
      provinceCode: '01',
      districtCode: '001',
      wardCode: '00001',
      streetLine: '123 Main St',
      fullAddress: '123 Main St, Hanoi',
      isDefault: false,
      type: AddressType.HOME,
    };
    const address = { id: 'address-1', profileId: profile.id, isDefault: true };

    prisma.address.count.mockResolvedValue(0);
    prisma.address.create.mockResolvedValue(address);

    await expect(service.addAddress(profile.userId, dto)).resolves.toEqual(
      address,
    );
    expect(prisma.address.create).toHaveBeenCalledWith({
      data: {
        profileId: profile.id,
        contactName: dto.contactName,
        phone: dto.phone,
        countryCode: 'VN',
        provinceCode: dto.provinceCode,
        districtCode: dto.districtCode,
        wardCode: dto.wardCode,
        streetAddress: dto.streetLine,
        fullAddress: dto.fullAddress,
        type: dto.type,
        isDefault: true,
      },
    });
  });

  it('clears the previous default before adding a new default address', async () => {
    prisma.address.count.mockResolvedValue(1);
    prisma.address.create.mockResolvedValue({ id: 'address-2' });

    await service.addAddress(profile.userId, {
      contactName: 'Jane Doe',
      phone: '0987654321',
      provinceCode: '79',
      districtCode: '760',
      wardCode: '26734',
      streetLine: '456 Second St',
      type: AddressType.OFFICE,
      isDefault: true,
    });

    expect(prisma.address.updateMany).toHaveBeenCalledWith({
      where: { profileId: profile.id },
      data: { isDefault: false },
    });
  });

  it('rejects an address that does not belong to the profile', async () => {
    prisma.address.findFirst.mockResolvedValue(null);

    await expect(
      service.getAddressById(profile.userId, 'address-404'),
    ).rejects.toThrow(NotFoundException);
  });

  it('sets the default address atomically', async () => {
    const address = {
      id: 'address-1',
      profileId: profile.id,
      isDefault: false,
    };
    const updated = { ...address, isDefault: true };
    prisma.address.findFirst.mockResolvedValue(address);
    prisma.$transaction.mockImplementation(async (callback) =>
      callback({
        address: {
          updateMany: jest.fn().mockResolvedValue({ count: 1 }),
          update: jest.fn().mockResolvedValue(updated),
        },
      }),
    );

    await expect(
      service.setDefaultAddress(profile.userId, address.id),
    ).resolves.toEqual(updated);
    expect(prisma.$transaction).toHaveBeenCalledTimes(1);
  });

  it('promotes another address after deleting the current default', async () => {
    const current = {
      id: 'address-1',
      profileId: profile.id,
      isDefault: true,
    };
    const next = {
      id: 'address-2',
      profileId: profile.id,
      isDefault: false,
    };
    prisma.address.findFirst
      .mockResolvedValueOnce(current)
      .mockResolvedValueOnce(next);
    prisma.address.delete.mockResolvedValue(current);
    prisma.address.update.mockResolvedValue({ ...next, isDefault: true });

    await service.deleteAddress(profile.userId, current.id);

    expect(prisma.address.delete).toHaveBeenCalledWith({
      where: { id: current.id },
    });
    expect(prisma.address.update).toHaveBeenCalledWith({
      where: { id: next.id },
      data: { isDefault: true },
    });
  });

  it('rejects a shop name already owned by another user', async () => {
    prisma.shopConfig.findFirst.mockResolvedValue({ id: 'shop-2' });

    await expect(
      service.registerShop(profile.userId, { shopName: 'Taken Shop' }),
    ).rejects.toThrow(ConflictException);
  });

  it('creates a shop and returns the refreshed profile', async () => {
    const refreshed = {
      ...profile,
      shopConfig: { id: 'shop-1', shopName: 'Test Shop' },
    };
    prisma.shopConfig.findFirst.mockResolvedValue(null);
    prisma.shopConfig.create.mockResolvedValue(refreshed.shopConfig);
    prisma.profile.findUnique
      .mockResolvedValueOnce(profile)
      .mockResolvedValueOnce(refreshed);

    await expect(
      service.registerShop(profile.userId, {
        shopName: 'Test Shop',
        description: 'A test shop',
      }),
    ).resolves.toEqual(refreshed);
    expect(prisma.shopConfig.create).toHaveBeenCalledWith({
      data: {
        profileId: profile.id,
        shopName: 'Test Shop',
        description: 'A test shop',
        logoUrl: '',
      },
    });
  });
});
