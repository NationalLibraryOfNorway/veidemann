import { Component, OnInit, ChangeDetectionStrategy, inject } from '@angular/core';
import {Observable, Subject, throwError} from 'rxjs';
import {catchError} from 'rxjs/operators';
import {MatDialog} from '@angular/material/dialog';
import {AbilityServiceSignal} from "@casl/angular";
import {MongoAbility} from '@casl/ability';
import {ControllerApiService, ErrorService} from '../../../core';
import {CrawlerStatus} from '../../../shared/models/controller/controller.model';
import {CrawlerStatusDialogComponent} from '../crawlerstatus-dialog/crawlerstatus-dialog.component';
import {AsyncPipe} from '@angular/common';
import {CrawlerStatusComponent} from '../crawlerstatus/crawlerstatus.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
  imports: [
    AsyncPipe,
    CrawlerStatusComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true
})
export class DashboardComponent implements OnInit {
  private errorService = inject(ErrorService);
  private controllerApiService = inject(ControllerApiService);
  private dialog = inject(MatDialog);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);

  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  updateRunStatus = new Subject<void>();
  crawlerStatus$: Observable<CrawlerStatus>;

  constructor() {
    this.can = this.abilityService.can;
  }

  ngOnInit(): void {
    this.crawlerStatus$ = this.controllerApiService.getCrawlerStatus().pipe(
      catchError(error => {
        this.errorService.dispatch(error);
        return throwError(error);
      })
    );
    this.updateRunStatus.next();
  }

  onChangeRunStatus(shouldPause: boolean) {
    this.dialog.open(CrawlerStatusDialogComponent, {
      disableClose: true,
      autoFocus: true,
      data: {shouldPause}
    }).afterClosed().subscribe(changeStatus => {
      if (changeStatus) {
        if (shouldPause) {
          this.controllerApiService.pauseCrawler();
        } else {
          this.controllerApiService.unpauseCrawler();
        }
        this.updateRunStatus.next();
      }
    });
  }


}
