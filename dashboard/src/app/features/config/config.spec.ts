import {ComponentFixture, TestBed} from '@angular/core/testing';
import {ActivatedRoute, provideRouter} from '@angular/router';
import {of} from 'rxjs';

import {OptionsService} from './services';
import {ConfigComponent} from './config';

describe('ConfigComponent navigation shell', () => {
  let fixture: ComponentFixture<ConfigComponent>;
  const optionsService = {next: vi.fn()};

  beforeEach(async () => {
    optionsService.next.mockClear();
    await TestBed.configureTestingModule({
      imports: [ConfigComponent],
      providers: [
        provideRouter([]),
        {provide: OptionsService, useValue: optionsService},
        {
          provide: ActivatedRoute,
          useValue: {
            data: of({options: {}}),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders the configuration content shell', () => {
    expect(fixture.nativeElement.querySelector('mat-drawer')).toBeNull();
    expect(fixture.nativeElement.querySelector('.section-shell > .section-content')).not.toBeNull();
  });

  it('provides resolved options to child configuration pages', () => {
    expect(optionsService.next).toHaveBeenCalledWith({});
  });
});
