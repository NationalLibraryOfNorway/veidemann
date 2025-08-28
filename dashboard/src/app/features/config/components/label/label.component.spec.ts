import {ComponentFixture, TestBed} from '@angular/core/testing';

import {LabelComponent} from './label.component';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {MaterialModule} from '../../../commons/material.module';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {DragDropModule} from '@angular/cdk/drag-drop';
import {LabelService} from '../../services/label.service';
import {of} from 'rxjs';
import { provideZonelessChangeDetection } from '@angular/core';

describe('LabelsComponent', () => {
  let component: LabelComponent;
  let fixture: ComponentFixture<LabelComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [LabelComponent],
      imports: [MaterialModule, FormsModule, ReactiveFormsModule, DragDropModule, NoopAnimationsModule],
      providers: [
        provideZonelessChangeDetection(),
        {
          provide: LabelService,
          useValue: {
            getLabelKeys: () => of([])
          }
        }
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(LabelComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
