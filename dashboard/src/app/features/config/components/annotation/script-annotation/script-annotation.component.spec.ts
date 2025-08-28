import {ComponentFixture, TestBed} from '@angular/core/testing';

import {UntypedFormBuilder} from '@angular/forms';
import {CoreTestingModule} from '../../../../core/core.testing.module';
import {ScriptAnnotationComponent} from './script-annotation.component';

describe('ScriptAnnotationComponent', () => {
  let component: ScriptAnnotationComponent;
  let fixture: ComponentFixture<ScriptAnnotationComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CoreTestingModule.forRoot()],
      providers: [
        UntypedFormBuilder,
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
