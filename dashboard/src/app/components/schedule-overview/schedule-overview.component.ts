import {
  ChangeDetectionStrategy,
  ChangeDetectorRef,
  Component,
  OnDestroy,
  OnInit,
  ViewChild,
  ViewEncapsulation
} from '@angular/core';
import {FullCalendarComponent} from '@fullcalendar/angular';
import {forkJoin, Subject} from 'rxjs';
import {ConfigObject, Kind} from '../../../shared/models';
import {ConfigApiService, ErrorService} from '../../../modules/core/services';
import {createListRequest} from '../../../modules/config/func/query';
import {takeUntil, toArray} from 'rxjs/operators';
import { parseCronExpression } from 'cron-schedule';
import {MatDialog} from '@angular/material/dialog';
import {ScheduleEventDialogComponent} from '../schedule-event-dialog/schedule-event-dialog.component';
import {colorScales} from './colors';
import {DateClickArg} from '@fullcalendar/interaction';
import {CalendarOptions, EventClickArg} from '@fullcalendar/core';
import dayGridPlugin from '@fullcalendar/daygrid';
import timeGridPlugin from '@fullcalendar/timegrid';
import interactionPlugin from '@fullcalendar/interaction';
import { endOfYear, isAfter, isBefore, parseISO, startOfYear } from 'date-fns';

interface ScheduledJob {
  crawlJobName: string;
  id: string;
  executionDates: {
    start: string,
    end: string,
  }[];
}

interface ScheduleValidRange {
  validFrom: string;
  validTo: string;
}

@Component({
    selector: 'app-schedule-overview',
    templateUrl: './schedule-overview.component.html',
    styleUrls: ['./schedule-overview.component.css'],
    providers: [ConfigApiService],
    changeDetection: ChangeDetectionStrategy.OnPush,
    encapsulation: ViewEncapsulation.Emulated,
    standalone: false
})
export class ScheduleOverviewComponent implements OnInit, OnDestroy {
  private crawlJobs: ConfigObject[];
  private crawlSchedules: ConfigObject[];
  private viewDate: Date = new Date();
  private ngUnsubscribe: Subject<void>;

  readyToLoad = false;
  calendarOptions: CalendarOptions;

  @ViewChild('scheduleCalendar') calendar: FullCalendarComponent;

  constructor(private errorService: ErrorService,
              private configApiService: ConfigApiService,
              private dialog: MatDialog,
              private cdr: ChangeDetectorRef) {

    this.ngUnsubscribe = new Subject<void>();

    this.calendarOptions = {
      eventClick: this.onEventClick.bind(this),
      initialView: 'dayGridMonth',
      plugins: [
        dayGridPlugin,
        timeGridPlugin,
        interactionPlugin
      ],
      headerToolbar: {
        start: 'today,prev,next',
        center: 'title',
        end: 'dayGridMonth,timeGridWeek,timeGridDay'
      },
      customButtons: {
        prev: {
          text: '<',
          click: this.onPrevious.bind(this)
        },
        next: {
          text: '>',
          click: this.onNext.bind(this)
        },
        today: {
          text: 'today',
          click: this.onToday.bind(this)
        }
      },
      dateClick: this.onDateClick.bind(this),
      height: 'auto',
      locale: 'NO-nb',
      validRange: (nowDate) => {
        return {
          start: new Date(nowDate.getFullYear(), nowDate.getMonth(), 1)
        };
      }
    };
  }

  ngOnDestroy() {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  ngOnInit(): void {


    forkJoin([
      this.configApiService.list(createListRequest(Kind.CRAWLJOB.valueOf())).pipe(toArray()),
      this.configApiService.list(createListRequest(Kind.CRAWLSCHEDULECONFIG.valueOf())).pipe(toArray()),
    ]).pipe(
      takeUntil(this.ngUnsubscribe),
    )
      .subscribe(([jobs, schedules]) => {
          this.crawlJobs = jobs.filter(configObject => configObject.crawlJob.disabled === false)
            .sort((a, b) => a.meta.name.localeCompare(b.meta.name));
          this.crawlSchedules = schedules;
          setTimeout(() => {
            this.updateCalendar();
          }, 150);
        },
        error => {
          this.errorService.dispatch(error);
        }
      );
  }

  private updateCalendar() {
    const scheduledJobs = this.getScheduledJobs();
    const events = [];
    const bc = colorScales.mode('rgb').colors(scheduledJobs.length);
    for (const [index, job] of scheduledJobs.entries()) {
      for (const interval of job.executionDates) {
        events.push({
          title: job.crawlJobName,
          start: interval.start,
          end: interval.end,
          crawlJobId: job.id,
          backgroundColor: bc[index],
        });
      }
    }
    this.calendarOptions.events = events;
    this.cdr.markForCheck();
  }

  private getScheduledJobs(): ScheduledJob[] {
    const scheduledJobs = [];

    for (const job of this.crawlJobs) {
      const scheduleRefId = job.crawlJob.scheduleRef.id;
      if (scheduleRefId === '') {
        continue;
      }
      const crawlSchedule = this.crawlSchedules.find(_ => _.id === scheduleRefId);
      if (crawlSchedule === undefined) {
        continue;
      }
      const cronExpression = crawlSchedule.crawlScheduleConfig.cronExpression;
      if (cronExpression === '') {
        continue;
      }
      const validRange: ScheduleValidRange = {
        validFrom: crawlSchedule.crawlScheduleConfig.validFrom,
        validTo: crawlSchedule.crawlScheduleConfig.validTo
      };

      const schedule = this.getScheduleFromCron(cronExpression, validRange, job.crawlJob.limits.maxDurationS);

      scheduledJobs.push({
        crawlJobName: job.meta.name,
        id: job.id,
        executionDates: schedule
      });
    }
    return scheduledJobs;
  }

  private getScheduleFromCron(cronExpression: string, validRange: ScheduleValidRange, durationS: number): { start: string, end: string }[] {
    const checkRange = validRange.validFrom || validRange.validTo ? true : false;
    const cron = parseCronExpression(cronExpression);
    const schedule: { start: string, end: string }[] = [];
    try {
      let startDate = new Date(this.viewDate.getFullYear(), this.viewDate.getMonth(), 1);
      let endDate = new Date(this.viewDate.getFullYear(), this.viewDate.getMonth() + 1, 1);
      for (const nextDate of cron.getNextDatesIterator(startDate, endDate)) {
          if (checkRange) {
            if (this.isDateInRange(nextDate, validRange)) {
              schedule.push({
                start: nextDate.toISOString(),
                end: this.addDurationS(nextDate, durationS).toISOString(),
              });
            }
          } else {
            schedule.push({
              start: nextDate.toISOString(),
              end: this.addDurationS(nextDate, durationS).toISOString(),
            });
          }

      }

      startDate = new Date(this.viewDate.getFullYear(), this.viewDate.getMonth(), 1);
      endDate = this.viewDate;
      for (const prevDate of cron.getPrevDatesIterator(startDate, endDate)) {
          if (checkRange) {
            // @ts-ignore
            if (this.isDateInRange(prevDate, validRange)) {
              schedule.push({
                // @ts-ignore
                start: obj.value.toISOString(),
                // @ts-ignore
                end: this.addDuration(prevDate, duration),
              });
            }
          } else {
            schedule.push({
              // @ts-ignore
              start: obj.value.toISOString(),
              // @ts-ignore
              end: this.addDuration(obj.value, duration),
            });
          }
      }
    } catch (err) {
      this.errorService.dispatch(err);
    }
    return schedule;
  }


  private onEventClick(event: EventClickArg) {
    const data = {
      id: event.event.extendedProps['crawlJobId'],
      start: event.event.startStr,
      end: event.event.endStr,
      name: event.event.title
    };

    this.dialog.open(ScheduleEventDialogComponent, {data});
  }

  private addDurationS(date: Date, durationS: number): Date {
    date.setSeconds(date.getSeconds() + durationS);
    return date
  }

  private onToday() {
    this.viewDate = new Date();
    this.calendar.getApi().today();
    this.updateCalendar();
  }

  private onNext() {
    this.calendar.getApi().next();
    if (this.calendar.getApi().getDate().getMonth() !== this.viewDate.getMonth()) {
      this.viewDate.setMonth(this.viewDate.getMonth() + 1);
      this.updateCalendar();
    }
  }

  private onPrevious() {
    this.calendar.getApi().prev();
    if (this.calendar.getApi().getDate().getMonth() !== this.viewDate.getMonth()) {
      this.viewDate.setMonth(this.viewDate.getMonth() - 1);
      this.updateCalendar();
    }
  }

  private onDateClick(cal: DateClickArg) {
    this.calendar.getApi().changeView('timeGridDay');
    this.calendar.getApi().gotoDate(cal.date);
  }

  private isDateInRange(startDate: Date, validRange: ScheduleValidRange) {
    const validFrom = validRange.validFrom ? parseISO(validRange.validFrom) : startOfYear(new Date());
    const validTo = validRange.validTo ? parseISO(validRange.validTo) : endOfYear(new Date());
    return isAfter(startDate, validFrom) && isBefore(startDate, validTo);
  }
}
