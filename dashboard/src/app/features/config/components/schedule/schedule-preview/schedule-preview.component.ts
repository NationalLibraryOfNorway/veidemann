import {Component, Input} from '@angular/core';
import {ConfigObject} from '../../../../../shared/models/config';
import {DatePipe} from '@angular/common';
import {MatLabel} from '@angular/material/form-field';
import {LayoutDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-schedule-preview',
  templateUrl: './schedule-preview.component.html',
  styleUrls: ['./schedule-preview.component.css'],
  imports: [
    DatePipe,

    LayoutDirective,
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
