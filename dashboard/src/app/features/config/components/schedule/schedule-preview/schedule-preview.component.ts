import {Component, Input} from '@angular/core';
import {ConfigObject} from '../../../../../shared/models/config';
import {DatePipe} from '@angular/common';
import {MatLabel} from '@angular/material/form-field';

@Component({
  selector: 'app-schedule-preview',
  templateUrl: './schedule-preview.component.html',
  styleUrls: ['./schedule-preview.component.css'],
  imports: [
    DatePipe,
    MatLabel
  ],
  standalone: true
})
export class SchedulePreviewComponent {
  @Input()
  configObject: ConfigObject;

  constructor() {
  }
}
