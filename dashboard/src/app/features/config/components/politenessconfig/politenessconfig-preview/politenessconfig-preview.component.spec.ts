import {ComponentFixture, TestBed} from '@angular/core/testing';

import {PolitenessconfigPreviewComponent} from './politenessconfig-preview.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('PolitenessconfigPreviewComponent', () => {
  let component: PolitenessconfigPreviewComponent;
  let fixture: ComponentFixture<PolitenessconfigPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [PolitenessconfigPreviewComponent],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PolitenessconfigPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.POLITENESSCONFIG});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
