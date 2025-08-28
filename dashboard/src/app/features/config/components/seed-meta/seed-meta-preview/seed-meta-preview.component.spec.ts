import {ComponentFixture, TestBed} from '@angular/core/testing';

import {SeedMetaPreviewComponent} from './seed-meta-preview.component';
import {CommonsModule} from '../../../../commons';
import {CoreTestingModule} from '../../../../core/core.testing.module';
import {ConfigObject, Kind} from '../../../../shared/models';

describe('SeedMetaPreviewComponent', () => {
  let component: SeedMetaPreviewComponent;
  let fixture: ComponentFixture<SeedMetaPreviewComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CoreTestingModule.forRoot(), CommonsModule],
      declarations: [SeedMetaPreviewComponent],
      providers: []
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(SeedMetaPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.SEED});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
