import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelMultiComponent} from './label-multi.component';
import {ConfigObject} from '../../../../../shared/models/config';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('LabelMultiComponent', () => {
  let component: LabelMultiComponent;
  let fixture: ComponentFixture<LabelMultiComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [LabelMultiComponent],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(LabelMultiComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
