import {ComponentFixture, TestBed} from '@angular/core/testing';
import {ActivatedRoute, convertToParamMap, provideRouter} from '@angular/router';
import {of} from 'rxjs';

import {OptionsService} from './services';
import {ConfigComponent} from './config';

describe('ConfigComponent navigation shell', () => {
  let fixture: ComponentFixture<ConfigComponent>;
  const optionsService = {next: vi.fn()};

  async function createComponent(detailId?: string): Promise<void> {
    await TestBed.configureTestingModule({
      imports: [ConfigComponent],
      providers: [
        provideRouter([]),
        {provide: OptionsService, useValue: optionsService},
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: of(convertToParamMap({kind: 'seed'})),
            data: of({options: {}}),
            firstChild: {
              snapshot: {paramMap: convertToParamMap(detailId ? {id: detailId} : {})},
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  }

  it('removes the configuration type drawer after a type is selected', async () => {
    await createComponent();

    expect(fixture.nativeElement.querySelector('mat-drawer')).toBeNull();
    expect(fixture.nativeElement.querySelector('.compact-title').textContent).toContain('Seed');
    expect(fixture.nativeElement.querySelector('.back-button').getAttribute('href')).toBe('/config');
  });

  it('links detail pages back to their deterministic type list', async () => {
    await createComponent('seed-id');

    expect(fixture.nativeElement.querySelector('.back-button').getAttribute('href')).toBe('/config/seed');
  });
});
