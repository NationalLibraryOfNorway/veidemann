import {ComponentFixture, TestBed} from '@angular/core/testing';

import {MetaPreviewComponent} from './meta-preview.component';
import {ConfigObject} from '../../../../../shared/models/config';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('MetaPreviewComponent', () => {
  let component: MetaPreviewComponent;
  let fixture: ComponentFixture<MetaPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [MetaPreviewComponent],
      providers: [
        ...provideCoreTesting
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(MetaPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
