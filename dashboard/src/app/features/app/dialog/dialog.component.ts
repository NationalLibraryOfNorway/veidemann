import { ChangeDetectionStrategy, Component, OnDestroy, OnInit, inject } from '@angular/core';
import {MatDialog, MatDialogRef} from '@angular/material/dialog';

import {Subject} from 'rxjs';
import {takeUntil} from 'rxjs/operators';

import {ErrorDialogComponent} from '../error-dialog/error-dialog.component';
import {ErrorService} from '../../../core';



@Component({
    selector: 'app-dialog',
    template: '',
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true
})
export class DialogComponent implements OnInit, OnDestroy {
  private dialog = inject(MatDialog);
  private errorService = inject(ErrorService);


  private ngUnsubscribe: Subject<void>;
  private dialogRef: MatDialogRef<ErrorDialogComponent>;

  constructor() {
    this.ngUnsubscribe = new Subject<void>();
  }

  ngOnInit(): void {
    this.errorService.error$
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe((error) => {
        // TODO: show all errors explicitly (not just the first)

        if (this.dialog.openDialogs.length > 0) {
          console.error(error);
        } else {
          this.dialogRef = this.dialog.open(ErrorDialogComponent, {data: {error}});
        }
      });
  }

  ngOnDestroy(): void {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
    if (this.dialogRef) {
      this.dialogRef.close();
    }
  }
}
