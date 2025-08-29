import {Component, OnInit} from '@angular/core';
import {Observable, Subject, throwError} from 'rxjs';
import {catchError} from 'rxjs/operators';
import {MatDialog} from '@angular/material/dialog';
import {AbilityService} from "@casl/angular";
import { ControllerApiService, ErrorService } from '../../../core';
import { CrawlerStatus } from '../../../shared/models/controller/controller.model';
import { CrawlerStatusDialogComponent } from '../crawlerstatus-dialog/crawlerstatus-dialog.component';
import {AsyncPipe} from '@angular/common';
import {CrawlerStatusComponent} from '../crawlerstatus/crawlerstatus.component';
import {FlexLayoutModule} from '@angular/flex-layout';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
  imports: [
    AsyncPipe,
    FlexLayoutModule,
    CrawlerStatusComponent
  ],
  standalone: true
})
export class DashboardComponent implements OnInit {
  readonly ability$: Observable<any>;

  updateRunStatus: Subject<void> = new Subject();
  crawlerStatus$: Observable<CrawlerStatus>;

  constructor(private errorService: ErrorService,
              private controllerApiService: ControllerApiService,
              private dialog: MatDialog,
              private abilityService: AbilityService<any>) {
    this.ability$ = this.abilityService.ability$;
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
