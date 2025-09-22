import {ComponentFixture, TestBed} from '@angular/core/testing';

import {AnnotationComponent} from './annotation.component';
import {UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('AnnotationComponent', () => {
  let component: AnnotationComponent;
  let fixture: ComponentFixture<AnnotationComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        UntypedFormBuilder,
        ...provideCoreTesting,
        {
          provide: AuthService,
          useValue: {
            isAdmin: () => true,
            canUpdate: () => true,
          }
        }
      ],
      declarations: [AnnotationComponent]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(AnnotationComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
