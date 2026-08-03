import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {AuthService} from '../../../../core';
import {Kind} from '../../../../shared/models';
import {ConfigNavListComponent} from './config-nav-list.component';

describe('ConfigNavListComponent', () => {
  let fixture: ComponentFixture<ConfigNavListComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConfigNavListComponent],
      providers: [
        provideRouter([]),
        {provide: AuthService, useValue: {isAdmin: () => false, isCurator: () => false}},
        {
          provide: AbilityServiceSignal,
          useValue: {can: (_action: string, subject: string) => subject === Kind[Kind.SEED]},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigNavListComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('shows authorized configuration types as destination tiles', () => {
    const links = fixture.nativeElement.querySelectorAll('.destination-link');
    expect(links.length).toBe(1);
    expect(links[0].getAttribute('href')).toBe('/config/seed');
    expect(links[0].textContent).toContain('Seed');
  });
});
