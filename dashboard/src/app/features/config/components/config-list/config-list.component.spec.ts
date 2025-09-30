import {ComponentFixture, TestBed} from '@angular/core/testing';
import {ConfigListComponent} from './config-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('ConfigListComponent', () => {
  let component: ConfigListComponent;
  let fixture: ComponentFixture<ConfigListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        ConfigListComponent,
      ],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ConfigListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
