import {Component, Inject, ChangeDetectionStrategy} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {RouterLink} from '@angular/router';
import {MatButtonModule} from '@angular/material/button';
import {DatePipe} from '@angular/common';

@Component({
  selector: 'app-schedule-event-dialog',
  templateUrl: './schedule-event-dialog.component.html',
  styleUrls: ['./schedule-event-dialog.component.css'],
  imports: [
    DatePipe,
    MatButtonModule,
    MatDialogModule,
    RouterLink
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  standalone: true
})
export class ScheduleEventDialogComponent {

  calendarEvent: any;

  constructor(@Inject(MAT_DIALOG_DATA) public data: any) {
    this.calendarEvent = data;
  }
}
