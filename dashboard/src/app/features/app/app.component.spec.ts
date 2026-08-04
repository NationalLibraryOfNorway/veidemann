import {Component} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {provideNoopAnimations} from '@angular/platform-browser/animations';
import {AbilityServiceSignal} from '@casl/angular';

import {AuthService, ErrorService, SnackBarService} from '../../core';
import {AppComponent} from './app.component';

@Component({template: '', standalone: true})
class EmptyRouteComponent {}

describe('AppComponent navigation', () => {
  let fixture: ComponentFixture<AppComponent>;
  let router: Router;
  const can = vi.fn((_action: string, subject: string) => subject === 'configs');

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideNoopAnimations(),
        provideRouter([
          {path: 'config', component: EmptyRouteComponent},
          {path: 'report', component: EmptyRouteComponent},
          {path: 'logconfig', component: EmptyRouteComponent},
        ]),
        {provide: AbilityServiceSignal, useValue: {can}},
        {
          provide: AuthService,
          useValue: {
            isLoggedIn: false,
            requestedUri: '',
            name: '',
            login: vi.fn(),
            logout: vi.fn(),
          },
        },
        {provide: ErrorService, useValue: {error$: {pipe: () => ({subscribe: vi.fn()})}, dispatch: vi.fn()}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn()}},
      ],
    }).compileComponents();

    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders only authorized primary destinations in the drawer', () => {
    expect(fixture.nativeElement.querySelector('mat-nav-list.mat-mdc-nav-list')).not.toBeNull();

    const links = fixture.nativeElement.querySelectorAll('a[href="/config"]');
    expect(links.length).toBe(1);
    expect(links[0].textContent).toContain('Configuration');
    expect(links[0].classList).toContain('mat-mdc-list-item');
    expect(links[0].querySelector('mat-icon').classList).toContain('mat-mdc-list-item-icon');
    expect(fixture.nativeElement.querySelector('a[href="/"]')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('a[href="/report"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('a[href="/logconfig"]')).toBeNull();
  });

  it('marks the active nested destination as the current page', async () => {
    await router.navigateByUrl('/config');
    fixture.detectChanges();
    await fixture.whenStable();

    const activeLinks = fixture.nativeElement.querySelectorAll('a[href="/config"][aria-current="page"]');
    expect(activeLinks.length).toBe(1);
    expect(activeLinks[0].classList).toContain('mdc-list-item--activated');
  });
});
