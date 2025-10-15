import {inject, TestBed} from '@angular/core/testing';
import {AuthService} from './auth.service';
import {OAuthService} from 'angular-oauth2-oidc';
import {provideCoreTesting} from '../core.testing.module';

describe('AuthService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        ...provideCoreTesting,
        {
          provide: AuthService,
          useValue: {
            isAdmin: () => true,
            canUpdate: () => true,
          }
        },
        {provide: OAuthService, useValue: {}}
      ]
    });
  });

  it('should be created', inject([AuthService], (service: AuthService) => {
    expect(service).toBeTruthy();
  }));
});
