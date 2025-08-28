import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelMultiComponent} from './label-multi.component';
import {LabelService} from '../../../services';
import {ConfigObject} from '../../../../shared/models/config';
import {MaterialModule} from '../../../../commons/material.module';
import { provideZonelessChangeDetection } from '@angular/core';

describe('LabelMultiComponent', () => {
  let component: LabelMultiComponent;
  let fixture: ComponentFixture<LabelMultiComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [MaterialModule],
      declarations: [LabelMultiComponent],
      providers: [
        provideZonelessChangeDetection(),
        {
          provide: LabelService,
          useValue: {}
        }
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
