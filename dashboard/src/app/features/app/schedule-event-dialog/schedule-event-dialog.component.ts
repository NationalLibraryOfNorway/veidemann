import { Component, ChangeDetectionStrategy, inject } from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {RouterLink} from '@angular/router';
import {MatButtonModule} from '@angular/material/button';
import {DatePipe} from '@angular/common';

interface ScheduleEventData {
  id: string;
  start: string;
  end: string;
  name: string;
}

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
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class ScheduleEventDialogComponent {
  readonly calendarEvent = inject<ScheduleEventData>(MAT_DIALOG_DATA);
}
