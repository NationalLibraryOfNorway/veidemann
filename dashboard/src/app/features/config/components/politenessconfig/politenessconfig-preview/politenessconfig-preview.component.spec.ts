import {ComponentFixture, TestBed} from '@angular/core/testing';

import {PolitenessconfigPreviewComponent} from './politenessconfig-preview.component';
import {DurationFormatPipe} from '../../../../../shared/pipes/duration-format.pipe';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {MatLabel} from '@angular/material/form-field';
import {MatChipListbox} from '@angular/material/chips';
import { provideZonelessChangeDetection } from '@angular/core';

describe('PolitenessconfigPreviewComponent', () => {
  let component: PolitenessconfigPreviewComponent;
  let fixture: ComponentFixture<PolitenessconfigPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [MatLabel, MatChipListbox],
      declarations: [PolitenessconfigPreviewComponent, DurationFormatPipe],
      providers: [
        provideZonelessChangeDetection(),
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
