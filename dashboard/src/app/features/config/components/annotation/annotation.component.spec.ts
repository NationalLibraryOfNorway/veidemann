import {ComponentFixture, TestBed} from '@angular/core/testing';

import {AnnotationComponent} from './annotation.component';
import {AuthService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('AnnotationComponent', () => {
  let component: AnnotationComponent;
  let fixture: ComponentFixture<AnnotationComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        AnnotationComponent
      ],
      providers: [
        ...provideCoreTesting,
        {
          provide: AuthService,
          useValue: {
            isAdmin: () => true,
            canUpdate: () => true,
          }
        }
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(AnnotationComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
