import {ComponentFixture, TestBed} from '@angular/core/testing';
import {LoglevelComponent} from './log';
import {of} from 'rxjs';
import {LogService} from './services';
import {LogLevel, Level} from '../../shared/models';
import {provideCoreTesting} from '../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';


describe('LoglevelComponent', () => {
  let component: LoglevelComponent;
  let fixture: ComponentFixture<LoglevelComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        LoglevelComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              data: {
                levels: [Level.TRACE, Level.DEBUG, Level.INFO, Level.WARN, Level.ERROR]
              }
            },
            params: of({}),
            queryParams: of({})
          }
        },
        {
          provide: LogService, useValue: {
            getLogConfig: () => of({logLevelList: [new LogLevel()]})
          }
        }
      ],

    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(LoglevelComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
