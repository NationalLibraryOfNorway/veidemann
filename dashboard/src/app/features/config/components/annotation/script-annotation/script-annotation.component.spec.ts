import {ComponentFixture, TestBed} from '@angular/core/testing';

import {UntypedFormBuilder} from '@angular/forms';
import {ScriptAnnotationComponent} from './script-annotation.component';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('ScriptAnnotationComponent', () => {
  let component: ScriptAnnotationComponent;
  let fixture: ComponentFixture<ScriptAnnotationComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        UntypedFormBuilder,
        ...provideCoreTesting
      ],
      declarations: [ScriptAnnotationComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ScriptAnnotationComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
