import {ComponentFixture, TestBed} from '@angular/core/testing';
import {LoglevelComponent} from './log';
import {of} from 'rxjs';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { CoreTestingModule } from '../../core/core.testing.module';
import { LogService } from './services';
import { LogLevel } from '../../shared/models';
import { AuthService } from '../../core';


describe('LoglevelComponent', () => {
  let component: LoglevelComponent;
  let fixture: ComponentFixture<LoglevelComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        NoopAnimationsModule,
        CoreTestingModule.forRoot()
      ],
      providers: [
        {
          provide: LogService, useValue: {
            getLogConfig: () => of({logLevelList: [new LogLevel()]})
          }
        },
        {
          provide: AuthService, useValue: {
            canUpdate: () => true
          }
        },
      ],
      declarations: [LoglevelComponent]
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
