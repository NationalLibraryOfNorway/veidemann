import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {AuthService} from '../../../../core';
import {ReportNavigationListComponent} from './report-navigation-list.component';

describe('ReportNavigationListComponent', () => {
  let fixture: ComponentFixture<ReportNavigationListComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ReportNavigationListComponent],
      providers: [
        provideRouter([]),
        {provide: AuthService, useValue: {isAdmin: () => false}},
        {
          provide: AbilityServiceSignal,
          useValue: {can: (_action: string, subject: string) => subject === 'pagelog'},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ReportNavigationListComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('shows authorized report types as destination tiles', () => {
    const links = fixture.nativeElement.querySelectorAll('.destination-link');
    expect(links.length).toBe(1);
    expect(links[0].getAttribute('href')).toBe('/report/pagelog');
    expect(links[0].textContent).toContain('Page log');
  });
});
